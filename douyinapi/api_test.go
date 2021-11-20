package douyinapi

import (
	"net/http"
	"testing"
)

func TestGenerateCookie(t *testing.T) {
	InitCookieGenerator("/home/ivan/.nvm/versions/node/v14.17.5/bin/node", "/home/ivan/workspace/dylive/douyinapi/acrawler.js")
	client := &http.Client{
		Timeout: HttpTimeout,
	}
	cookie, err := GenerateCookie(client, "https://live.douyin.com/hongjingzhibo")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cookie: ", cookie)
}

func TestGetUserIdByName(t *testing.T) {
	InitCookieGenerator("/home/ivan/.nvm/versions/node/v14.17.5/bin/node", "/home/ivan/workspace/dylive/douyinapi/acrawler.js")
	user, err := GetUserByName("hongjingzhibo")
	if err != nil {
		t.Fatal(err)
	}
	if user.SecUid != "MS4wLjABAAAAuw4X7CNDvaXlGM7HE-jp2jMtQC9U0lkICEE-Pg8i7AM" {
		t.Error("bad user secuid", user.SecUid)
	}
	if user.NickName != "红警直播舞虾" {
		t.Error("bad user nickname", user.NickName)
	}
	t.Log("user: ", user)
}
