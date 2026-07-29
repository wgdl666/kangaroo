package configcenter

// Bundle is the published Config Center payload (settings + secrets).
type Bundle struct {
	PSM         string            `json:"psm"`
	Environment string            `json:"environment"`
	ReleaseID   string            `json:"release_id"`
	Generation  int64             `json:"generation"`
	ETag        string            `json:"etag"`
	Settings    map[string]any    `json:"settings"`
	Secrets     map[string]string `json:"secrets"`
}

// Snapshot is the locally held published config after a successful fetch.
type Snapshot struct {
	Bundle       Bundle
	ETag         string
	RequestedEnv string // XX_WG_ENV used to address Config Center
	EffectiveEnv string // environment actually served (may be prod after PPE fallback)
	Fallback     bool   // true when PPE fell back to prod
}
