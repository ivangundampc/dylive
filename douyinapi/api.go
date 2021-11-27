package douyinapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	// Timeout for all HTTP requests
	HttpTimeout = 5 * time.Second

	// User-Agent header for all HTTP requests
	UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"
	Cookie    = ""

	NodePath   = ""
	ScriptPath = ""

	ErrInvalidData = errors.New("invalid page data")
	ErrNoSuchUser  = errors.New("no such user")
	ErrorNoUrl     = errors.New("No URL found")
	ErrorNoRoom    = errors.New("No room found")
)

type (
	Id uint64

	User struct {
		Id       Id     // will change over time
		UniqueId Id     // will change over time
		SecUid   string // constant
		Name     string // user id in user profile page
		NickName string // aka "display name"
		Picture  string // url of thumbnail picture
		Room     *Room  // live stream room (if any)
	}

	Room struct {
		Id        Id
		PageUrl   string
		Title     string
		Status    int
		Operating bool
		CreatedAt time.Time

		LikesCount              int
		CurrentUsersCount       int
		NewFollowersCount       int
		GiftsUniqueVisitorCount int
		FansCount               int
		TotalUsersCount         int

		StreamId        Id
		StreamHeight    int
		StreamWidth     int
		StreamHlsUrlMap map[string]string
	}
)

func (id Id) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(id), 10))
}

func InitCookieGenerator(nodePath string, scriptPath string) error {
	if _, err := os.Stat(nodePath); errors.Is(err, os.ErrNotExist) {
		return err
	}
	NodePath = nodePath
	if _, err := os.Stat(scriptPath); errors.Is(err, os.ErrNotExist) {
		return err
	}
	ScriptPath = scriptPath
	return nil
}

func GenerateCookie(client *http.Client, url string) (string, error) {
	// default go douyin homepage
	var req *http.Request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// set the header with the same user agent
	req.Header.Set("User-Agent", UserAgent)

	// execute the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// retrieve __ac_nonce from html
	__ac_nonce := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__ac_nonce" {
			__ac_nonce = cookie.Value
			fmt.Println("Found __ac_nonce:", __ac_nonce)
		}
	}

	// execute node program to calculate __ac_signature
	cmd := exec.Command(NodePath, ScriptPath, __ac_nonce)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("Cannot execute node program - error: %s", err.Error())
		return "", err
	}
	__ac_signature := string(stdout)

	// construct the cookie
	acNonceCookie := http.Cookie{Name: "__ac_nonce", Value: __ac_nonce}
	acSignCookie := http.Cookie{Name: "__ac_signature", Value: __ac_signature}
	acRefererCookie := http.Cookie{Name: "__ac_referer", Value: "__ac_blank"}

	var sb strings.Builder
	sb.WriteString(acNonceCookie.String())
	sb.WriteString(";")
	sb.WriteString(acSignCookie.String())
	sb.WriteString(";")
	sb.WriteString(acRefererCookie.String())

	return sb.String(), nil
}

func GetUserByName(name string) (user *User, err error) {

	client := &http.Client{
		Timeout: HttpTimeout,
	}

	url := "https://live.douyin.com/" + name
	if Cookie == "" {
		Cookie, err = GenerateCookie(client, url)
		fmt.Printf("Generate cookie: %s\n", Cookie)
		if err != nil {
			return nil, err
		}
	}

	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Cookie", Cookie)

	if err != nil {
		return nil, err
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var b []byte
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res, err := parseLivePageHtml(string(b))
	// error may due to cookie invalid, clear the cookie to generate at next time
	if err != nil {
		Cookie = ""
		return nil, err
	}

	return res, nil
}

type (
	livePageData struct {
		Location string `json:"location"`
		Odin     struct {
			UserID       string `json:"user_id"`
			UserUniqueID string `json:"user_unique_id"`
		} `json:"odin"`
		InitialState struct {
			RoomStore struct {
				RoomInfo struct {
					Room   *dyRoom `json:"room"`
					RoomID string  `json:"roomId"`
					Anchor struct {
						Nickname    string `json:"nickname"`
						AvatarThumb struct {
							URLList []string `json:"url_list"`
						} `json:"avatar_thumb"`
						SecUID string `json:"sec_uid"`
					} `json:"anchor"`
				} `json:"roomInfo"`
			} `json:"roomStore"`
		} `json:"initialState"`
		RouteInitialProps struct {
			ErrorType string `json:"errorType"`
		} `json:"routeInitialProps"`
	}

	dyRoom struct {
		IdString          string `json:"id_str"`
		Title             string `json:"title"`
		LikesCount        int    `json:"like_count"`
		CurrentUsersCount int    `json:"user_count"`
		Status            int    `json:"status"` // 2 - started, 4 - ended
		CreatedAt         int    `json:"create_time"`
		Stats             struct {
			NewFollowersCount       int `json:"follow_count"`
			GiftsUniqueVisitorCount int `json:"gift_uv_count"` // maybe
			FansCount               int `json:"fan_ticket"`    // maybe
			TotalUsersCount         int `json:"total_user"`
		} `json:"stats"`
		StreamUrl struct {
			IdString string `json:"id_str"`
			Extra    struct {
				Height int `json:"height"`
				Width  int `json:"width"`
			} `json:"extra"`
			HlsUrlMap map[string]string `json:"hls_pull_url_map"`
		} `json:"stream_url"`
	}
)

func (room *dyRoom) toRoom() *Room {
	if room == nil {
		return nil
	}
	return &Room{
		Id:                      strToId(room.IdString),
		PageUrl:                 roomUrl(room.IdString),
		Title:                   room.Title,
		Status:                  room.Status,
		Operating:               room.Status == 2,
		CreatedAt:               time.Unix(int64(room.CreatedAt), 0).UTC(),
		LikesCount:              room.LikesCount,
		CurrentUsersCount:       room.CurrentUsersCount,
		NewFollowersCount:       room.Stats.NewFollowersCount,
		GiftsUniqueVisitorCount: room.Stats.GiftsUniqueVisitorCount,
		FansCount:               room.Stats.FansCount,
		TotalUsersCount:         room.Stats.TotalUsersCount,
		StreamId:                strToId(room.StreamUrl.IdString),
		StreamHeight:            room.StreamUrl.Extra.Height,
		StreamWidth:             room.StreamUrl.Extra.Width,
		StreamHlsUrlMap:         room.StreamUrl.HlsUrlMap,
	}
}

func parseLivePageHtml(html string) (*User, error) {
	a := strings.Index(html, "RENDER_DATA")
	if a < 0 {
		return nil, ErrInvalidData
	}
	html = html[a:]
	a = strings.Index(html, ">")
	if a < 0 {
		return nil, ErrInvalidData
	}
	html = html[a+1:]
	a = strings.Index(html, "<")
	if a < 0 {
		return nil, ErrInvalidData
	}
	html = html[:a]
	html, err := url.QueryUnescape(html)
	if err != nil {
		return nil, err
	}
	var data livePageData
	err = json.Unmarshal([]byte(html), &data)
	if err != nil {
		return nil, err
	}
	if data.RouteInitialProps.ErrorType == "server-error" {
		return nil, ErrNoSuchUser
	}
	roomInfo := data.InitialState.RoomStore.RoomInfo
	var picture string
	pictures := roomInfo.Anchor.AvatarThumb.URLList
	if len(pictures) > 0 {
		picture = pictures[0]
	}
	return &User{
		Id:       strToId(data.Odin.UserID),
		UniqueId: strToId(data.Odin.UserUniqueID),
		SecUid:   roomInfo.Anchor.SecUID,
		Room:     roomInfo.Room.toRoom(),
		Name:     strings.Trim(data.Location, "/"),
		NickName: roomInfo.Anchor.Nickname,
		Picture:  picture,
	}, nil
}

func roomUrl(id string) string {
	return "https://webcast.amemv.com/webcast/reflow/" + id
}

func strToId(in string) Id {
	out, _ := strconv.ParseUint(in, 10, 64)
	return Id(out)
}

const (
	scriptOpen  = "<script>window.__INIT_PROPS__ = "
	scriptClose = "</script>"
)

func GetRoom(url string) (room *Room, err error) {
	if url == "" {
		err = ErrorNoUrl
		return
	}

	var resp *http.Response
	client := &http.Client{
		Timeout: HttpTimeout,
	}

	if Cookie == "" {
		Cookie, err = GenerateCookie(client, url)
		if err != nil {
			return
		}
	}

	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UserAgent)
	if err != nil {
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var b []byte
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	room = getRoomFromHtml(string(b))
	if room == nil {
		err = ErrorNoRoom
	}
	return
}

func getRoomFromHtml(html string) *Room {
	i := strings.Index(html, scriptOpen)
	if i < 0 {
		return nil
	}
	html = html[i+len(scriptOpen):]
	i = strings.Index(html, scriptClose)
	if i < 0 {
		return nil
	}
	html = html[:i]
	bytes := []byte(html)
	var obj map[string]map[string]dyRoom
	json.Unmarshal(bytes, &obj)
	room := obj["/webcast/reflow/:id"]["room"]
	if room.IdString != "" {
		return room.toRoom()
	}
	return nil
}
