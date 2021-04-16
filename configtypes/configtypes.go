package configtypes

type DefaultScribe []string

type ScRotationDescriptor struct {
	RpKeepFilesNum  int `json:"rpKeepFilesNum"`
	RpLogLimitBytes int `json:"rpLogLimitBytes"`
	RpMaxAgeHours   int `json:"rpMaxAgeHours"`
}

type SetupScribeDescriptor struct {
	ScFormat   string                `json:"scFormat"`
	ScKind     string                `json:"scKind"`
	ScMaxSev   string                `json:"scMaxSev"`
	ScMinSev   string                `json:"scMinSev"`
	ScName     string                `json:"scName"`
	ScPrivacy  string                `json:"scPrivacy"`
	ScRotation *ScRotationDescriptor `json:"scRotation"`
}

type LogBufferBkCfg struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
