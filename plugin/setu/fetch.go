package setu

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type Pic struct {
	Pid    int      `json:"pid"`
	Title  string   `json:"title"`
	Tags   []string `json:"tags"`
	Author string   `json:"author"`
	Uid    int      `json:"uid"`
	Urls   struct {
		Original string `json:"original"`
	} `json:"urls"`
}

const apiUrl = "https://api.lolicon.app/setu/v2"

// 使用Tag来获取图片
func FetchOnlinWithTags(tags []string) ([]Pic, error) {
	data := map[string]interface{}{
		"tag": tags,
	}
	dataJson, _ := json.Marshal(data)
	reqBody := bytes.NewReader(dataJson)

	req, _ := http.NewRequest("POST", apiUrl, reqBody)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respData struct {
		Error string `json:"error"`
		Data  []Pic  `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return nil, err
	}

	// 替换为镜像站
	for i := range respData.Data {
		p := &respData.Data[i]
		p.Urls.Original = strings.Replace(p.Urls.Original, "i.pixiv.cat", "i.pixiv.re", 1)
	}

	return respData.Data, nil
}

// 随机获取图片
func FetchOnlineRandom() ([]Pic, error) {
	return FetchOnlinWithTags(nil)
}
