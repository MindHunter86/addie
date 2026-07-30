package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
)

type (
	Title struct {
		// Names      interface{}
		// Updated    uint64
		// LastChange uint64 `json:"last_change"`

		Id     uint16
		Code   string
		Player *Player
	}
	Player struct {
		// AlternativePlayer string `json:"alternative_player"`
		// Series            interface{}

		Host     string
		Playlist map[string]*PlayerPlaylist
	}
	PlayerPlaylist struct {
		// CreatedTimestamp uint64 `json:"created_timestamp"`
		// Preview          string
		// Skips            interface{}

		Serie uint16
		Hls   *PlaylistHls
	}
	PlaylistHls struct {
		Fhd, Hd, Sd string
	}
)

type (
	TitleSerie struct {
		Title, Serie  uint16
		QualityHashes map[utils.TitleQuality]string
	}
)

// redis schema
// https://cache.libria.fun/videos/media/ts/9277/13/720/3ae5aa5839690b8d9ea9fcef9b720fb4_00028.ts
// https://cache.libria.fun/videos/media/ts/9222/11/1080/97d3bb428727bc25fa110bc51826a366.m3u8
// KEY - VALUE
// ID:SERIE - FHD:HD:SD (links)
// ID:SERIE:{FHD,HD,SD} - json{[TS LINKS]}

const (
	tsrRawTitleId = uint8(iota) + 4 // 4 is a skipping of "/videos/media/ts" parts
	tsrRawTitleSerie
	tsrRawTitleQuality
	tsrRawFilename
)

func getQualityHash(rawpath string) (string, bool) {
	if rawpath == "" {
		return "", false
	}

	filename := strings.Split(rawpath, "/")[tsrRawFilename]
	return getHashFromUriPath(filename)
}

func getHashFromUriPath(upath string) (hash string, ok bool) {
	switch upath[len(upath)-1:] {
	case "s": // .ts
		if hash, _, ok = strings.Cut(upath, ".ts"); !ok {
			return "", ok
		}

		if hash, _, ok = strings.Cut(hash, "_"); !ok {
			return "", ok
		}
	case "8": // .m3u8
		if hash, _, ok = strings.Cut(upath, ".m3u8"); !ok {
			return "", ok
		}
	default:
		return "", false
	}

	return hash, ok
}

func (m *App) getTitleSerieFromCache(c *fiber.Ctx, tsr *TitleSerieRequest) (*TitleSerie, bool) {
	serie, e := m.cache.PullSerie(c, tsr.getTitleId(), tsr.getSerieId())
	if e != nil {
		gLog.Warn().Err(e).Msg("")
		return nil, false
	}

	return serie, serie != nil
}

func (m *App) getTitleSeriesFromApi(c *fiber.Ctx, titleId string) (_ []*TitleSerie, e error) {
	var title *Title
	e = gAniApi.getApiResponse(c, http.MethodGet, apiMethodGetTitle,
		[]string{"id", titleId}).parseApiResponse(&title)

	if e != nil {
		return nil, e
	}

	return m.validateTitleFromApiResponse(title), e
}

func (*App) validateTitleFromApiResponse(title *Title) (tss []*TitleSerie) {
	for _, serie := range title.Player.Playlist {
		if serie == nil {
			gLog.Warn().Msg("there is an empty serie found in the api response's playlist")
			continue
		} else if serie.Hls == nil {
			gLog.Warn().Msg("there is an empty serie.HLS found in the api response's playlist")
			continue
		} else if serie.Hls.Sd == "" && serie.Hls.Hd == "" && serie.Hls.Fhd == "" {
			gLog.Warn().Msg("internal error; serie quality is defined but empty")
			continue
		}

		tserie, ok := &TitleSerie{}, false
		tserie.Title, tserie.Serie = title.Id, serie.Serie
		tserie.QualityHashes = make(map[utils.TitleQuality]string)

		if tserie.QualityHashes[utils.TitleQualitySD], ok = getQualityHash(serie.Hls.Sd); !ok {
			gLog.Warn().Uint16("tid", tserie.Title).Uint16("sed", tserie.Serie).Msg("there is no SD quality for parsed title")
		}

		if tserie.QualityHashes[utils.TitleQualityHD], ok = getQualityHash(serie.Hls.Hd); !ok {
			gLog.Warn().Uint16("tid", tserie.Title).Uint16("sed", tserie.Serie).Msg("there is no HD quality for parsed title")
		}

		if tserie.QualityHashes[utils.TitleQualityFHD], ok = getQualityHash(serie.Hls.Fhd); !ok {
			gLog.Warn().Uint16("tid", tserie.Title).Uint16("sed", tserie.Serie).Msg("there is no FHD quality for parsed title")
		}

		tss = append(tss, tserie)
	}

	return
}

func (m *App) doTitleSerieRequest(c *fiber.Ctx, tsr *TitleSerieRequest) (ts *TitleSerie, e error) {
	var ok bool

	rlog(c).Debug().Str("tid", tsr.getTitleIdString()).Str("sid", tsr.getSerieIdString()).Msg("trying to get series from cache")
	if ts, ok = m.getTitleSerieFromCache(c, tsr); ok {
		return
	}

	var tss []*TitleSerie
	rlog(c).Info().Str("tid", tsr.getTitleIdString()).Str("sid", tsr.getSerieIdString()).Msg("trying to get series from api")
	if tss, e = m.getTitleSeriesFromApi(c, tsr.getTitleIdString()); e != nil {
		return
	}

	if len(tss) == 0 {
		return nil, errors.New("there is an empty result in the response")
	}

	for _, t := range tss {
		if t.Serie == tsr.getSerieId() {
			ts = t
		}

		if e = m.cache.PushSerie(t); e != nil {
			rlog(c).Warn().Err(e).Str("tid", tsr.getTitleIdString()).Str("sid", tsr.getSerieIdString()).Msg("")
			continue
		}
	}

	if ts == nil {
		return nil, errors.New("could not find requesed serie id in the response")
	}

	return
}

func (m *App) getUriWithFakeQuality(c *fiber.Ctx, tsr *TitleSerieRequest, uri string, quality utils.TitleQuality) string {
	rlog(c).Debug().Msg("format check")
	if tsr.isOldFormat() && !tsr.isM3U8() {
		rlog(c).Info().Str("old", "/"+tsr.getTitleQualityString()+"/").Str("new", "/"+quality.String()+"/").Str("uri", uri).Msg("format is old")
		return strings.ReplaceAll(uri, "/"+tsr.getTitleQualityString()+"/", "/"+quality.String()+"/")
	}

	rlog(c).Debug().Msg("trying to complete tsr")
	title, e := m.doTitleSerieRequest(c, tsr)
	if e != nil {
		rlog(c).Error().Err(e).Msg("could not rewrite quality for the request")
		return uri
	}

	rlog(c).Debug().Msg("trying to get hash")
	hash, ok := tsr.getTitleHash()
	if !ok {
		return uri
	}

	rlog(c).Debug().Str("old_hash", hash).Str("new_hash", title.QualityHashes[quality]).Str("uri", uri).Msg("")
	return strings.ReplaceAll(
		strings.ReplaceAll(uri, "/"+tsr.getTitleQualityString()+"/", "/"+quality.String()+"/"),
		hash, title.QualityHashes[quality],
	)
}

type TitleSerieRequest struct {
	titleId, serieId uint16
	quality          utils.TitleQuality
	hash             string

	raw []string
}

func NewTitleSerieRequest(uri string) *TitleSerieRequest {
	return &TitleSerieRequest{
		raw: strings.Split(uri, "/"),
	}
}

func (m *TitleSerieRequest) getTitleId() uint16 {
	if m.titleId != 0 {
		return m.titleId
	}

	tid, _ := strconv.ParseUint(m.raw[tsrRawTitleId], 10, 16)
	m.titleId = uint16(tid)
	return m.titleId
}

func (m *TitleSerieRequest) getTitleIdString() string {
	return m.raw[tsrRawTitleId]
}

func (m *TitleSerieRequest) getSerieId() uint16 {
	if m.serieId != 0 {
		return m.serieId
	}

	sid, _ := strconv.ParseUint(m.raw[tsrRawTitleSerie], 10, 16)
	m.serieId = uint16(sid)
	return m.serieId
}

func (m *TitleSerieRequest) getSerieIdString() string {
	return m.raw[tsrRawTitleSerie]
}

func (m *TitleSerieRequest) getTitleQuality() utils.TitleQuality {
	if m.quality != utils.TitleQualityNone {
		return m.quality
	}

	switch m.raw[tsrRawTitleQuality] {
	case "480":
		m.quality = utils.TitleQualitySD
		return m.quality
	case "720":
		m.quality = utils.TitleQualityHD
		return m.quality
	case "1080":
		m.quality = utils.TitleQualityFHD
		return m.quality
	default:
		return utils.TitleQualityNone
	}
}

func (m *TitleSerieRequest) getTitleQualityString() string {
	return m.raw[tsrRawTitleQuality]
}

func (m *TitleSerieRequest) getTitleHash() (_ string, ok bool) {
	if m.hash != "" {
		return m.hash, true
	}

	m.hash, ok = getHashFromUriPath(m.raw[tsrRawFilename])
	return m.hash, ok
}

// TODO - refactor
func (m *TitleSerieRequest) isOldFormat() bool {
	return strings.Contains(m.raw[tsrRawFilename], "fff")
}

func (m *TitleSerieRequest) isM3U8() bool {
	return strings.Contains(m.raw[tsrRawFilename], "m3u8")
}

func (m *TitleSerieRequest) isValid() bool {
	return len(m.raw) == 8
}
