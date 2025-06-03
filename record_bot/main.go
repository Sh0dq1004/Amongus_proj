package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	//not neccesary
	//"reflect"

	"github.com/bwmarrin/discordgo"
)

var (
	file *os.File
	vc_r *discordgo.VoiceConnection
	vc_p *discordgo.VoiceConnection
)

func main(){
	var token string
	fmt.Println("録音用botのトークンを入力してください")
	fmt.Scanln(&token)
	record_dg := bot_init(token)
	fmt.Println("再生用botのトークンを入力してください")
	fmt.Scanln(&token)
	player_dg := bot_init(token)

	defer record_dg.Close()
	defer player_dg.Close()

	record_dg.AddHandler(Command4Recorder)
	player_dg.AddHandler(Command4Player)

	record_dg.Identify.Intents=discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent
	player_dg.Identify.Intents=discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent

	err:=record_dg.Open()
	if err != nil{
		fmt.Println("接続エラー in recorder:", err)
		return
	}
	err=player_dg.Open()
	if err != nil{
		fmt.Println("接続エラー in player:", err)
		return
	}

	fmt.Println("Bot起動中。!start recorder で録音開始。")
	fmt.Println("Bot起動中。!start player で録音開始。")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("全てのBotを終了します。")
}

func bot_init(token string) (dg *discordgo.Session){
	dg, err := discordgo.New("Bot "+token);
	if err!=nil{
		fmt.Println("Bot作成エラー:", err)
		return
	}
	return
}

func Command4Recorder(s *discordgo.Session, m *discordgo.MessageCreate){
	var err error
	if m.Author.Bot{return}

	if m.Content=="!start record"{
		vc_r, err=connectVC(m.GuildID, m.Author.ID, s, m)
		if err!=nil{
			fmt.Println("recorder not started yet")
		}
		go startRecord(s, m)
	}

	if m.Content=="!stop record"{
		file.Close()
		vc_r.Disconnect()
	}
	
}

func Command4Player(s *discordgo.Session, m *discordgo.MessageCreate){
	var err error
	if m.Author.Bot{return}

	if m.Content=="!start player"{
		vc_p, err=connectVC(m.GuildID, m.Author.ID, s, m)
		if err!=nil{
			fmt.Println("recorder not started yet")
		}
		go startRecord(s, m)
	}

	if m.Content=="!stop player"{
		vc_p.Disconnect()
	}
}

func startRecord(s *discordgo.Session, m *discordgo.MessageCreate){
	var err error
	s.ChannelMessageSend(m.ChannelID, "録音を開始します。")

	vc_r.Speaking(true)

	file, err = os.Create("record.opus")
	if err != nil{
		fmt.Println("ファイル作成エラー:",err)
		return
	}
	for {
		p, ok := <-vc_r.OpusRecv
		if !ok{
			fmt.Println("音声受信終了")
			break
		}
		file.Write(p.Opus)
	}
}

func connectVC(guildID string, userID string, s *discordgo.Session, m *discordgo.MessageCreate) (vc *discordgo.VoiceConnection, err error){
	var channelID string
	guild,_:=s.State.Guild(guildID)
	for _, vs:=range guild.VoiceStates{
		if vs.UserID==userID{
			channelID=vs.ChannelID
			break
		}
	}

	if channelID==""{
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルに参加してからコマンドを送ってください。")
		return
	}

	vc, err = s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err!=nil{
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルへの接続に失敗しました。")
		fmt.Println("vc接続エラー:", err)
		return
	}
	return
}
