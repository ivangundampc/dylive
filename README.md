# dylive

Get live stream m3u8 URL and monitor users for new live broadcasts.

You can:
- Watch (multiple) Douyin HD live streams in your favorite video player on
  your desktop (without phone)
- Write your own script and get notified (or start recording, etc.) once new
  live stream is started

NOTE: dysearch is not working any more currently.

## Install

```
go get -v -u github.com/caiguanhao/dylive

# if you want to search for Douyin users, also install dysearch
go get -v -u github.com/caiguanhao/dylive/dysearch
```

Recently, Douyin added an anti-crawler on their live pages that sends "__ac_nonce" and requires client to calculate "__ac_signature"
A node.js script is added in "acrawler" to generate "__ac_signature". Extra installation steps is required:
```shell
# Assume we clone this repo to "dylive"
cd dylive/acrawler
npm install
```

Before running any api in douyinapi, you need to do an initialization
```go
err = douyinapi.InitCookieGenerator("/usr/bin/node", "/home/user/dylive/acrawler/acrawler.js")
if err != nil {
  log.Panic(err)
}
monitorUser, _ := douyinapi.GetUserByName(monitorUserId)
```

## Usage

### Search And Watch

Search top 2 Douyin live streamers whose name contains 红警 (Red Alert) and
open mpv to watch their live streams.

```
dysearch -L -F -n 2 红警 | xargs dylive | xargs -n1 open -na mpv
```

`dysearch` supports printing results in a table.

```
dysearch -L -F -table 红警
ID                NAME              FOLLOWERS  FAVORITED  ROOM CREATED  VIEWERS  NICK NAME
4094182951237853  dbg666666666666   464790     4001151    1h57m45s      3851     红警直播大彬
64607696525       268509981         366973     193566     1h8m4s        1656     红警飞哥
86712476626       LaoSiJi666666888  179176     754590     1h12m54s      318      红警老撕鸡🐔
452608268184679   wkf2319           147923     1179171    2h48m48s      926      红警王尤里
93603545482       chaorenhongjing   120428     13476      1h46m0s       270      红警阳光超人
94792729333       hongjingzhibo     116173     230472     1m3s          4        红警直播舞虾
4221738880606076  hongjingniusan    79228      26814      9m58s         7597     红警直播牛三
97894106911       890835888         24556      26941      56m19s        78       红警程弟
59773964913       chashu666         7197       692        6h50m16s      30       红警直播老茶666
```

### Watch Live Stream

In any user profile page, copy user ID (user name) listed below user's nick name.

<img src="https://user-images.githubusercontent.com/1284703/124866056-59660200-dfee-11eb-8f98-05419cbe115f.jpg" width="400" />

Use the user ID as the argument of the command. For example:

```
dylive 1011694538 | xargs open -na mpv
```

You can use [streamlink](https://streamlink.github.io/) to download the live stream while watching it.

```
dylive 1011694538 | xargs -I X streamlink --player /Applications/mpv.app/Contents/MacOS/mpv -r video.ts X best
```

## Execute Command

You can use the `-exec` option to run a command, especially useful for Windows.

```
# play and record the live stream
dylive -exec "streamlink --player mpv -r video.ts {{.LiveStreamUrl}} best" ...

# ... with a custom file name
dylive -exec "streamlink -r {{printf \"%s - %s.ts\" .User.NickName \
  (.User.Room.CreatedAt.Format \"2006-01-02\") | printf \"%q\"}} {{.LiveStreamUrl}} best" ...
```

The command can read the live stream info in JSON from standard input.
For example, open multiple live streams and tile the windows:

```
dylive -exec "bash cmd.sh" list-of-ids...
```

```
# cmd.sh
info=($(jq -r ".Index%4%2*100, .Index/2%2*100, .LiveStreamUrl, .User.NickName"))
x=${info[0]}
y=${info[1]}
url=${info[2]}
name=${info[3]}
mpv --really-quiet --title="$name" --no-border --geometry="50%+$x%+$y%" $url &
```

https://user-images.githubusercontent.com/1284703/122740688-cc653e00-d2b6-11eb-86a8-0bffb9e33a7a.mp4

## API

```go
import "github.com/caiguanhao/dylive/douyinapi"
```

Docs: <https://pkg.go.dev/github.com/caiguanhao/dylive/douyinapi>
