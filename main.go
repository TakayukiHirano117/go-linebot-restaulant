package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

func init() {
	godotenv.Load()
}

func main() {
	// ハンドラの登録
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/callback", lineHandler)

	// RenderではPORT環境変数を使う必要がある
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル用のデフォルト
	}

	fmt.Printf("Server is running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Hello World!!!!"
	fmt.Fprintf(w, msg)
}

func lineHandler(w http.ResponseWriter, r *http.Request) {
	// 環境変数からLINEチャネル情報を取得
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")

	fmt.Println("Secret:", channelSecret)
	fmt.Println("Token:", channelToken)

	// ログで確認
	log.Printf("Using channel secret: %s", maskString(channelSecret))
	log.Printf("Using access token: %s", maskString(channelToken))

	// BOTを初期化
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Printf("Failed to create bot: %v", err)
		w.WriteHeader(500)
		return
	}

	// リクエストからBOTのイベントを取得
	events, err := bot.ParseRequest(r)
	if err != nil {
		log.Printf("Error parsing request: %v", err)
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	// イベントごとにログを出力
	for _, event := range events {
		log.Printf("Received event: %+v", event)

		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				log.Printf("Text message: %s", message.Text)
				replyMessage := message.Text
				_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
				if err != nil {
					log.Printf("Failed to reply: %v", err)
				}
			case *linebot.LocationMessage:
				log.Printf("Location received: lat=%f, lng=%f", message.Latitude, message.Longitude)
				sendRestoInfo(bot, event)
			default:
				log.Printf("Unhandled message type: %T", message)
			}
		}
	}
}

// 位置情報を元に緯度・経度を返信する
func sendRestoInfo(bot *linebot.Client, e *linebot.Event) {
	msg := e.Message.(*linebot.LocationMessage)

	lat := strconv.FormatFloat(msg.Latitude, 'f', 6, 64)
	lng := strconv.FormatFloat(msg.Longitude, 'f', 6, 64)

	replyMsg := fmt.Sprintf("緯度：%s\n経度：%s", lat, lng)
	log.Printf("Sending reply: %s", replyMsg)

	_, err := bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(replyMsg)).Do()
	if err != nil {
		log.Printf("Failed to send location reply: %v", err)
	}
}

// セキュリティのためにログ出力時に値をマスクする
func maskString(s string) string {
	if len(s) <= 6 {
		return "******"
	}
	return s[:3] + "..." + s[len(s)-3:]
}
