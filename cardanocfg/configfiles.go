package cardanocfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/sjson"

	"github.com/thedevsaddam/gojsonq"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"go.uber.org/zap"
)

const Testnet = "testnet"
const Mainnet = "mainnet"
const URI = "https://hydra.iohk.io/job/Cardano/cardano-node/cardano-deployment/latest-finished/download/1"
const FilePath = "/home/lovelace/cardano-node"
const ConfigJSON = "config.json"
const ByronGenesis = "byron-genesis.json"
const ShelleyGenesis = "shelley-genesis.json"
const TopologyJSON = "topology.json"

type Downloader struct {
	log  *zap.SugaredLogger
	conf *config.C
	node *config.Node
}

type Topology struct {
	Producers []Producer `json:"Producers"`
}

type Producer struct {
	Atype   string `json:"type"`
	Addr    string `json:"addr"`
	Port    uint   `json:"port"`
	Valency uint   `json:"valency"`
	Debug   string `json:"debug"`
}

func New(n *config.Node, c *config.C) (*Downloader, error) {
	var err error
	d := &Downloader{}
	d.conf = c
	d.node = n
	if d.log, err = l.NewLogConfig(c, "config"); err != nil {
		return d, err
	}

	return d, nil
}

func (d *Downloader) GetFilePath(aType string, isTmp bool) (filePath string, err error) {
	filePath = fmt.Sprintf("%s/config/%s-%s",
		d.node.TmpDir,
		d.node.Network,
		aType)

	if !isTmp {
		filePath = fmt.Sprintf("%s/config/%s-%s",
			d.node.RootDir,
			d.node.Network,
			aType)
	}

	dir := filepath.Dir(filePath)
	if _, er := os.Stat(dir); er != nil {
		if os.IsNotExist(er) {
			if er := os.MkdirAll(dir, os.ModePerm); er != nil {
				return filePath, er
			}
		}
	}
	return filePath, err
}

func (d *Downloader) GetURL(aType string) (url string, err error) {
	url = fmt.Sprintf("%s/%s-%s", URI, d.node.Network, aType)
	return url, err
}

func (d *Downloader) GetConfigFile(aType string) (filePath string, err error) {
	var url string

	if filePath, err = d.GetFilePath(aType, false); err != nil {
		return filePath, err
	}

	statInfo, err := os.Stat(filePath)

	if err == nil {
		fTime := statInfo.ModTime()
		eDuration := time.Since(fTime).Hours()
		d.log.Info("Duration: ", eDuration)
		if eDuration < 24*5 {
			d.log.Info("file is recent: ", filePath)
		}
	}

	switch aType {
	case ConfigJSON:
		var filePathTmp string
		if filePathTmp, err = d.GetFilePath(aType, true); err != nil {
			return filePath, err
		}

		if url, err = d.GetURL(aType); err != nil {
			return filePath, err
		}
		if er := d.DownloadFile(filePathTmp, url); er != nil {
			return filePath, er
		}

		JSONBytes, er := ioutil.ReadFile(filePathTmp)
		if er != nil {
			return filePath, er
		}

		newJSON, er := sjson.SetBytes(JSONBytes, "hasPrometheus", []interface{}{"0.0.0.0", 12798})
		if er != nil {
			return filePath, er
		}

		mapBackEnd := make(map[string]interface{})

		keys := []string{
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

		for _, key := range keys {
			switch key {
			case "cardano.node.metrics":
				mapBackEnd[key] = []string{"TraceForwarderBK", "EKGViewBK"}
			case "cardano.node.resources":
				mapBackEnd[key] = []string{"TraceForwarderBK", "EKGViewBK"}
			case "cardano.node.IpSubscription":
				mapBackEnd[key] = []string{"TraceForwarderBK", "KatipBK"}
			// case "cardano.node.Handshake":
			//	mapBackEnd[key] = []string{"TraceForwarderBK"}
			default:
				// mapBackEnd[key] = []string{"TraceForwarderBK", "KatipBK"}
				d.log.Warn("no default trace for: ", key)
			}
		}

		if newJSON, err = sjson.SetBytes(newJSON, "options.mapBackends", mapBackEnd); err != nil {
			return filePath, err
		}

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

		if newJSON, err = sjson.SetBytes(newJSON, "mapSubtrace.KKKekgview.contents", contentsx); err != nil {
			return filePath, err
		}

		if newJSON, err = sjson.SetBytes(newJSON, "mapSubtrace.ekgview.subtrace", "FilterTrace"); err != nil {
			return filePath, err
		}

		if newJSON, err = sjson.SetBytes(newJSON, "mapSubtrace.cardanoXXXepoch-validationXXXutxo-stats", "NoTrace"); err != nil {
			return filePath, err
		}

		if newJSON, err = sjson.SetBytes(newJSON, "mapSubtrace.cardanXXXnode-metrics", "Neutral"); err != nil {
			return filePath, err
		}

		traceForwardToMap := make(map[string]interface{})
		contents := []string{"monitor", fmt.Sprintf("%d", d.node.RtViewPort)}
		traceForwardToMap["tag"] = "RemoteSocket"
		traceForwardToMap["contents"] = contents

		if newJSON, err = sjson.SetBytes(newJSON, "traceForwardTo", traceForwardToMap); err != nil {
			return filePath, err
		}

		traceMempool := false
		if d.node.IsProducer {
			traceMempool = true
		}

		if newJSON, err = sjson.SetBytes(newJSON, "TraceMempool", traceMempool); err != nil {
			return filePath, err
		}

		d.log.Infof("setting min severity for node %d to %s", d.node.Network, d.node.LogMinSeverity)
		if newJSON, err = sjson.SetBytes(newJSON, "minSeverity", d.node.LogMinSeverity); err != nil {
			return filePath, err
		}

		traces := []string{
			//"TraceBlockFetchClient",
			"TraceBlockFetchDecisions",
			"TraceBlockFetchProtocol",
			"TraceBlockFetchProtocolSerialised",
			"TraceBlockFetchServer",
			"TraceHandshake",
		}

		for _, trace := range traces {
			if newJSON, err = sjson.SetBytes(newJSON, trace, true); err != nil {
				return filePath, err
			}
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
			return filePath, err
		}

		d.log.Info("created file: ", filePath)

	case ByronGenesis:
		if url, err = d.GetURL(aType); err != nil {
			return filePath, err
		}

		err = d.DownloadFile(filePath, url)
	case ShelleyGenesis:
		if url, err = d.GetURL(aType); err != nil {
			return filePath, err
		}
		err = d.DownloadFile(filePath, url)

		jq := gojsonq.New().File(filePath)

		d.node.NetworkMagic = uint64(jq.From("networkMagic").Get().(float64))
		d.log.Infof("node %s network magic: %d", d.node.Name, d.node.NetworkMagic)

	case TopologyJSON:
		err = d.DownloadAndSetTopologyFile()
	}

	return filePath, err
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (d *Downloader) DownloadFile(filePath, url string) error {
	d.log.Info("downloading from URL: ", url)
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(resp)
		fmt.Println(err.Error())
		return err
	}
	defer resp.Body.Close()

	// Create the file
	d.log.Info("creatind dir: ", filepath.Dir(filePath))
	if er := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); er != nil {
		return er
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	d.log.Info("saved to file: ", filePath)

	return err
}
