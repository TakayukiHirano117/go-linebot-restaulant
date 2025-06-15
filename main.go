package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

func main() {
	godotenv.Load()

	// ハンドラの登録
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/callback", lineHandler)

	fmt.Println("http://localhost:8080 で起動中...")
	// HTTPサーバを起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Hello World!!!!"
	fmt.Fprintf(w, msg)
}

func lineHandler(w http.ResponseWriter, r *http.Request) {
	// BOTを初期化
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// リクエストからBOTのイベントを取得
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		// イベントがメッセージの受信だった場合
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// メッセージがテキスト形式の場合
			case *linebot.TextMessage:
				replyMessage := message.Text
				_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
				if err != nil {
					log.Print(err)
				}
			// メッセージが位置情報の場合
			case *linebot.LocationMessage:
				sendRestoInfo(bot, event)
			}
			// 他にもスタンプや画像、位置情報など色々受信可能
		}
	}
}

func sendRestoInfo(bot *linebot.Client, e *linebot.Event) {
	msg := e.Message.(*linebot.LocationMessage)

	lat := strconv.FormatFloat(msg.Latitude, 'f', 2, 64)
	lng := strconv.FormatFloat(msg.Longitude, 'f', 2, 64)

	replyMsg := getRestoInfo(lat, lng)

	_, err := bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(replyMsg)).Do()
	if err != nil {
		log.Print(err)
	}
}

// response APIレスポンス
type response struct {
	Results results `json:"results"`
}

// results APIレスポンスの内容
type results struct {
	Shop []shop `json:"shop"`
}

// shop レストラン一覧
type shop struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func getRestoInfo(lat string, lng string) string {
	apikey := os.Getenv("HOTPEPPER_API_KEY")
	if apikey == "" {
		log.Printf("Error: HOTPEPPER_API_KEY is not set")
		return "APIキーが設定されていません"
	}

	url := fmt.Sprintf(
		"https://webservice.recruit.co.jp/hotpepper/gourmet/v1/?format=json&key=%s&lat=%s&lng=%s&range=3&count=5",
		apikey, lat, lng)
	
	log.Printf("Requesting HotPepper API: %s", url)

	// リクエストしてボディを取得
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error making request to HotPepper API: %v", err)
		return "レストラン情報の取得に失敗しました"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HotPepper API returned non-200 status code: %d", resp.StatusCode)
		return "レストラン情報の取得に失敗しました"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "レストラン情報の取得に失敗しました"
	}

	log.Printf("Received response from HotPepper API: %s", string(body))

	var data response
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error parsing JSON response: %v", err)
		return "レストラン情報の解析に失敗しました"
	}

	if len(data.Results.Shop) == 0 {
		return "周辺にレストランが見つかりませんでした"
	}

	info := "周辺のレストラン情報：\n\n"
	for _, shop := range data.Results.Shop {
		info += fmt.Sprintf("店舗名：%s\n住所：%s\n\n", shop.Name, shop.Address)
	}
	return info
}