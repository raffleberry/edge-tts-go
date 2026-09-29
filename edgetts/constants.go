package edgetts

const (
	BaseURL            = "speech.platform.bing.com/consumer/speech/synthesize/readaloud"
	TrustedClientToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"

	WSSURL    = "wss://" + BaseURL + "/edge/v1?TrustedClientToken=" + TrustedClientToken
	VoiceList = "https://" + BaseURL + "/voices/list?trustedclienttoken=" + TrustedClientToken

	DefaultVoice = "en-US-EmmaMultilingualNeural"

	ChromiumFullVersion        = "143.0.3650.75"
	ChromiumMajorVersion       = "143"
	SecMSGECVersion            = "1-" + ChromiumFullVersion
	UserAgent                  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0"
	AcceptEncoding             = "gzip, deflate, br, zstd"
	AcceptLanguage             = "en-US,en;q=0.9"
	WSSOrigin                  = "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"
	VoiceAuthority             = "speech.platform.bing.com"
	VoiceSecCHUA               = `" Not;A Brand";v="99", "Microsoft Edge";v="143", "Chromium";v="143"`
	VoiceSecCHUAMobile         = "?0"
	VoiceAccept                = "*/*"
	VoiceSecFetchSite          = "none"
	VoiceSecFetchMode          = "cors"
	VoiceSecFetchDest          = "empty"
	TicksPerSecond       int64 = 10_000_000
	MP3BitrateBPS        int64 = 48_000
)

// BaseHeaders returns the base HTTP headers used by all requests.
func BaseHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      UserAgent,
		"Accept-Encoding": AcceptEncoding,
		"Accept-Language": AcceptLanguage,
	}
}

// WSSHeaders returns the headers used for the WebSocket handshake.
func WSSHeaders() map[string]string {
	h := BaseHeaders()
	h["Pragma"] = "no-cache"
	h["Cache-Control"] = "no-cache"
	h["Origin"] = WSSOrigin
	return h
}

// VoiceHeaders returns the headers used for the voice list request.
func VoiceHeaders() map[string]string {
	h := BaseHeaders()
	h["Authority"] = VoiceAuthority
	h["Sec-CH-UA"] = VoiceSecCHUA
	h["Sec-CH-UA-Mobile"] = VoiceSecCHUAMobile
	h["Accept"] = VoiceAccept
	h["Sec-Fetch-Site"] = VoiceSecFetchSite
	h["Sec-Fetch-Mode"] = VoiceSecFetchMode
	h["Sec-Fetch-Dest"] = VoiceSecFetchDest
	return h
}
