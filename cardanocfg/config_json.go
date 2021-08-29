package cardanocfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/adakailabs/gocnode/downloader"

	"github.com/juju/errors"

	"github.com/tidwall/sjson"
)

var cardanoBlocks = []string{
	"cardano.node.metrics",
	"cardano.node.resources",
	"cardano.node.AcceptPolicy",
	"cardano.node.ChainDB",
	"cardano.node.DnsResolver",
	"cardano.node.DnsSubscription",
	"cardano.node.ErrorPolicy",
	"cardano.node.Handshake",
	"cardano.node.IpSubscription",
	"cardano.node.LocalErrorPolicy",
	"cardano.node.LocalHandshake",
	"cardano.node.Mux",
}

func (d *Downloader) ConfigureBlocks(jsonB []byte) ([]byte, error) {
	var err error
	mapBackEnd := make(map[string]interface{})

	for _, key := range cardanoBlocks {
		switch key {
		case "cardano.node.metrics":
			mapBackEnd[key] = []string{"TraceForwarderBK", "EKGViewBK"}
		case "cardano.node.resources":
			mapBackEnd[key] = []string{"TraceForwarderBK", "EKGViewBK"}
		case "cardano.node.IpSubscription":
			mapBackEnd[key] = []string{"TraceForwarderBK", "KatipBK"}
		// FIXME:
		// case "cardano.node.Handshake":
		// mapBackEnd[key] = []string{"TraceForwarderBK"}
		default:
			// FIXME: mapBackEnd[key] = []string{"TraceForwarderBK", "KatipBK"}
			d.log.Warn("no default trace for: ", key)
		}
	}

	jsonB, err = sjson.SetBytes(jsonB, "options.mapBackends", mapBackEnd)
	if err != nil {
		return jsonB, errors.Annotatef(err, "while configuring blocks")
	}

	jsonB, err = sjson.SetBytes(jsonB, "defaultBackends", []string{"TraceForwarderBK", "KatipBK"})
	if err != nil {
		return jsonB, errors.Annotatef(err, "while configuring blocks")
	}

	return jsonB, err
}

func (d *Downloader) SetEKGVIEWContents(jsonB []byte) ([]byte, error) {
	var err error
	type ContentsInner struct {
		Contains string `json:"contains"`
		Tag      string `json:"tag"`
	}

	contents0 := []interface{}{
		ContentsInner{"cardano.epoch-validation.benchmark", "Contains"},
		[]ContentsInner{{".monoclock.basic.", "Contains"}},
	}
	contents1 := []interface{}{
		ContentsInner{"cardano.epoch-validation.benchmark", "Contains"},
		[]ContentsInner{{"diff.RTS.cpuNs.timed.", "Contains"}},
	}
	contents2 := []interface{}{
		ContentsInner{"cardano.epoch-validation.benchmark", "Contains"},
		[]ContentsInner{{"", "Contains"}},
	}
	contents3 := []interface{}{
		ContentsInner{"#ekgview.#aggregation.cardano.epoch-validation.benchmark", "StartsWith"},
		[]ContentsInner{{"diff.RTS.gcNum.timed.", "Contains"}},
	}

	contentsx := []interface{}{
		contents0, contents1, contents2, contents3,
	}

	jsonB, err = sjson.SetBytes(jsonB, "mapSubtrace.KKKekgview.contents", contentsx)
	if err != nil {
		err = errors.Annotatef(err, "while configuring blocks")
	}
	return jsonB, err
}

func (d *Downloader) SetMapSubtraceContents(jsonB []byte) ([]byte, error) {
	var err error
	if jsonB, err = sjson.SetBytes(jsonB, "mapSubtrace.ekgview.subtrace", "FilterTrace"); err != nil {
		return jsonB, err
	}

	if jsonB, err = sjson.SetBytes(jsonB, "mapSubtrace.cardanoXXXepoch-validationXXXutxo-stats", "NoTrace"); err != nil {
		return jsonB, err
	}

	if jsonB, err = sjson.SetBytes(jsonB, "mapSubtrace.cardanXXXnode-metrics", "Neutral"); err != nil {
		return jsonB, err
	}

	return jsonB, err
}

func (d *Downloader) SetTraceForwardTo(newJSON []byte) ([]byte, error) {
	var err error
	traceForwardToMap := make(map[string]interface{})
	contents := []string{"monitor", fmt.Sprintf("%d", d.node.RtViewPort)}

	traceForwardToMap["tag"] = "RemoteSocket"
	traceForwardToMap["contents"] = contents

	if newJSON, err = sjson.SetBytes(newJSON, "traceForwardTo", traceForwardToMap); err != nil {
		return newJSON, err
	}
	return newJSON, err
}

func (d *Downloader) SetTraceMempool(newJSON []byte) ([]byte, error) {
	var err error

	traceMempool := false
	if d.node.IsProducer {
		traceMempool = true
	}

	if newJSON, err = sjson.SetBytes(newJSON, "TraceMempool", traceMempool); err != nil {
		return newJSON, err
	}

	return newJSON, err
}

func (d *Downloader) SetTraces(newJSON []byte) ([]byte, error) {
	var err error
	d.log.Infof("setting min severity for node %s to %s", d.node.Network, d.node.LogMinSeverity)
	if newJSON, err = sjson.SetBytes(newJSON, "minSeverity", d.node.LogMinSeverity); err != nil {
		return newJSON, err
	}

	traces := []string{
		// "TraceBlockFetchClient",
		"TraceBlockFetchDecisions",
		"TraceBlockFetchProtocol",
		"TraceBlockFetchProtocolSerialised",
		"TraceBlockFetchServer",
		"TraceHandshake",
	}

	for _, trace := range traces {
		if newJSON, err = sjson.SetBytes(newJSON, trace, true); err != nil {
			return newJSON, err
		}
	}

	return newJSON, err
}

func (d *Downloader) SetPrometheus(newJSON []byte) ([]byte, error) {
	var err error

	newJSON, er := sjson.SetBytes(newJSON, "hasPrometheus", []interface{}{"0.0.0.0", 12798})
	if er != nil {
		return newJSON, er
	}

	return newJSON, err
}

func (d *Downloader) GetConfigJSON(aType string) (filePath string, err error) {
	var filePathTmp string
	var url string
	if filePathTmp, err = d.GetFilePath(aType, true); err != nil {
		return filePath, err
	}

	if filePath, err = d.GetFilePath(aType, false); err != nil {
		return filePath, err
	}

	if url, err = d.GetURL(aType); err != nil {
		return filePath, err
	}
	if er := downloader.DownloadFile(filePathTmp, url); er != nil {
		return filePath, er
	}

	newJSON, er := ioutil.ReadFile(filePathTmp)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetPrometheus(newJSON)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.ConfigureBlocks(newJSON)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetEKGVIEWContents(newJSON)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetMapSubtraceContents(newJSON)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetTraceForwardTo(newJSON) // here
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetTraceMempool(newJSON)
	if er != nil {
		return filePath, er
	}

	newJSON, er = d.SetTraces(newJSON)
	if er != nil {
		return filePath, er
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, newJSON, "", "  ")
	if err != nil {
		return filePath, err
	}

	JSONString := prettyJSON.String()
	JSONString = strings.Replace(JSONString, "XXX", ".", -1)
	JSONString = strings.Replace(JSONString, "KKK", "#", 1)

	err = ioutil.WriteFile(filePath, []byte(JSONString), os.ModePerm)
	if err != nil {
		err = errors.Annotatef(err, "writing to: %s", filePath)
		return filePath, err
	}

	d.log.Info("created file: ", filePath)

	return filePath, err
}
