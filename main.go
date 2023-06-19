package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	DiscordToken string `json:"discord_token"`
	OpenAIKey    string `json:"openai_api_key"`
	OpenAIModel  string `json:"openai_model"`
}

var (
	config        Config
	gameText      string
	isGamePlaying bool
)

func main() {
	// Config dosyasını yükle
	loadConfig()

	// Discord oturumunu oluştur
	dg, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatal("Discord oturumu oluşturulamadı:", err)
	}

	// Botun çalıştığından emin olmak için ready event'ini dinle
	dg.AddHandler(ready)

	// Mesajları dinlemek için messageCreate event'ini dinle
	dg.AddHandler(messageCreate)

	// Botu etkinleştir
	err = dg.Open()
	if err != nil {
		log.Fatal("Discord oturumu açılamadı:", err)
	}

	fmt.Println("Bot çalışıyor. CTRL-C ile durdurun.")

	// Bot durumunu güncelle
	err = updateBotStatus(dg)
	if err != nil {
		log.Fatal("Bot durumu güncellenirken bir hata oluştu:", err)
	}

	// CTRL-C ile botu durdurmak için bir sinyal bekle
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Botu kapat
	dg.Close()
}

func updateBotStatus(s *discordgo.Session) error {
	// Botun durumu sürekli güncellenmesi için bir goroutine başlat
	go func() {
		for {
			if isGamePlaying {
				s.UpdateGameStatus(0, gameText)
			} else {
				s.UpdateGameStatus(0, "")
			}
			time.Sleep(60 * time.Second) // Durumu her 60 saniyede bir güncelle
		}
	}()

	return nil
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	// Bot hazır olduğunda yapılacak işlemler
	fmt.Println("Bot hazır!")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // Botun kendi mesajlarını işlemlemesini önle
	}

	if strings.HasPrefix(m.Content, "!setgame") {
		// Oyun metnini al
		gameText = strings.TrimPrefix(m.Content, "!setgame ")
		if gameText != "" {
			isGamePlaying = true
			s.ChannelMessageSend(m.ChannelID, "Oyun metni güncellendi: "+gameText)
		} else {
			isGamePlaying = false
			s.ChannelMessageSend(m.ChannelID, "Oyun metni temizlendi.")
		}
	}
	if strings.HasPrefix(m.Content, "!f") {
		// Duyuru metnini al
		announcement := strings.TrimPrefix(m.Content, "!f")

		// Duyuru yap
		response, err := openAIChatCompletion(announcement)
		if err != nil {
			log.Println("Yapay zeka yanıtı oluşturulamadı:", err)
			return
		}

		// İlk yanıtı al
		answer := response.Choices[0].Message.Content
		// Embed mesaj oluştur
		embed := &discordgo.MessageEmbed{
			Title:       "Fedai-YapayZeka Destekli Discord Botu:",
			Color:       0x00ff00, // Embed rengi (Yeşil)
			Description: answer,
		}

		// Embed mesajını gönder
		s.ChannelMessageSendEmbed(m.ChannelID, embed)

		// Yapay zeka yanıtını duyuru olarak gönder

	}
}

func openAIChatCompletion(message string) (*ChatCompletionResponse, error) {
	url := "https://api.openai.com/v1/chat/completions"
	data := map[string]interface{}{
		"model": config.OpenAIModel,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			{
				"role":    "user",
				"content": message,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.OpenAIKey)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chatCompletionResponse ChatCompletionResponse
	err = json.NewDecoder(resp.Body).Decode(&chatCompletionResponse)
	if err != nil {
		return nil, err
	}

	return &chatCompletionResponse, nil
}

type ChatCompletionResponse struct {
	ID        string           `json:"id"`
	Object    string           `json:"object"`
	Model     string           `json:"model"`
	Choices   []ChoiceResponse `json:"choices"`
	Complete  bool             `json:"complete"`
	Totals    TotalsResponse   `json:"totals"`
	Created   int64            `json:"created"`
	Usage     UsageResponse    `json:"usage"`
	Prompt    string           `json:"prompt"`
	MaxTokens int              `json:"max_tokens"`
}

type ChoiceResponse struct {
	Message MessageResponse `json:"message"`
}

type MessageResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TotalsResponse struct {
	Choices int `json:"choices"`
	Turns   int `json:"turns"`
}

type UsageResponse struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func loadConfig() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal("Config dosyası yüklenemedi:", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatal("Config dosyası ayrıştırılamadı:", err)
	}
}
