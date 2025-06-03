package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	//not neccesary
	"reflect"

	"github.com/bwmarrin/discordgo"
)

type Bot struct{
	Token string
}

func main(){
	Token := "MTM3ODk4Njg1NDE0OTUyNTUzNA.GTuuhN.-m5W5104PYJB96vcBeM7a0vBctGevPtH2Fst-g"

	dg, err:=discordgo.New("Bot "+Token)
	t := reflect.typeOf(dg)
	fmt.Println(t)
	if err != nil{
		fmt.Println("Bot作成エラー:", err)
		return
	}

	dg.AddHandler(onMessageCreate)

	dg.Identify.Intents=discordgo.IntentsGuildMessage|discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent

	err=dg.Open()
	if err != nil{
		fmt.Println("接続エラー:", err)
		return
	}

	fmt.Println("Bot起動中。!connect recorder で録音開始。")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Botを終了します。")
	dg.Close()

}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate){
	if m.Author.Bot{return}

	if m.Content=="!connect recorder"{
		startRecord(m.GuildID, m.Author.ID, s, m)
	}
	
}

func startRecord(guildID string, userID string, s *discordgo.Session, m *discordgo.MessageCreate){
	var channelID string
	guild,_:=s.State.Guild(guildID)
	for _, vs:=range guild.VoiceStates{
		if vs.userID==userID{
			channelID=vs.ChannelID
			break
		}
	}

	if channelID==""{
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルに参加してからコマンドを送ってください。")
		return
	}

	vc, err:=s.ChannelVoiceJoin(guildID, channelID, true, true)
	if err!=nil{
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルへの接続に失敗しました。")
		fmt.Println("vs接続エラー:", err)
		return
	}

	s.ChannelMessageSend(m.ChannelID, "録音を開始します。")

	vc.Speaking(true)

	file, err := os.Create("record.opus")
	if err != nil{
		fmt.Println("ファイル作成エラー:",err)
		return
	}

	go func(){
		defer file.Close()
		for {
			p, ok := <-vc.OpusRecv
			if !ok{
				fmt.Println("音声受信終了")
				break
			}
			file.Write(p.Opus)
		}
	}()
}
