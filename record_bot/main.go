package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"layeh.com/gopus"
	//"time"
	//"container/list"

	//not neccesary
	//"reflect"

	"github.com/bwmarrin/discordgo"
)

var (
	file *os.File
	vc_r *discordgo.VoiceConnection
	vc_p *discordgo.VoiceConnection
	sound_data []chan *discordgo.Packet
	encoder, _ = gopus.NewEncoder(48000, 2, gopus.Audio)
	decorder_list = []*gopus.Decoder{
		gopus.NewDecoder(48000, 2), //hontai,err=gopus.NewDecoder(48000, 2) だからエラー
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
		gopus.NewDecoder(48000, 2),
	}
	user_list [] uint32
	playing [] uint32
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

	fmt.Println("Bot起動中。!start record で録音開始。")
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
	var err_bool bool = false
	if m.Author.Bot{return}

	if m.Content=="!start record"{
		vc_r, err, err_bool= connectVC(m.GuildID, m.Author.ID, s, m)
		if err!=nil||err_bool{
			fmt.Println("recorder not started yet")
			return
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
	var err_bool bool = false
	if m.Author.Bot{return}

	if m.Content=="!start player"{
		vc_p, err, err_bool = connectVC(m.GuildID, m.Author.ID, s, m)
		if err!=nil||err_bool{
			fmt.Println("recorder not started yet")
			return 
		}
		go startPlayer(s, m)
	}

	if m.Content=="!stop player"{
		vc_p.Disconnect()
	}
}

func connectVC(guildID string, userID string, s *discordgo.Session, m *discordgo.MessageCreate) (vc *discordgo.VoiceConnection, err error, err_bool bool){
	var channelID string
	err_bool=false
	guild,err:=s.State.Guild(guildID)
	if err != nil || guild == nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			err_bool = true
			s.ChannelMessageSend(m.ChannelID, "ギルド情報の取得に失敗しました。")
			return
		}
	}
	for _, vs:=range guild.VoiceStates{
		if vs.UserID==userID{
			channelID=vs.ChannelID
			break
		}
	}
	if channelID==""{
		err_bool=true
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルに参加してからコマンドを送ってください。")
		return 
	}

	vc, err = s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err!=nil{
		err_bool=true
		s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルへの接続に失敗しました。")
		fmt.Println("vc接続エラー:", err)
		return
	}
	return
}

func startRecord(s *discordgo.Session, m *discordgo.MessageCreate){
	s.ChannelMessageSend(m.ChannelID, "録音を開始します。")
	vc_r.Speaking(true)
	for {
		p,_:= <-vc_r.OpusRecv
		pss:=p.SSRC
		exit,i:=find(user_list, pss)
		if !exit{
			pchan := make (chan *discordgo.Packet, 128)
			sound_data=append(sound_data, pchan)
			user_list=append(user_list, pss)
		}
		sound_data[i] <- p
	}
}

func startPlayer(s *discordgo.Session, m *discordgo.MessageCreate){
	s.ChannelMessageSend(m.ChannelID, "再生を開始します。")
	vc_p.Speaking(true)

	for {
		vc_p.OpusSend <- opusMixer()
	}

	/*
	for {
		for i,pchan := range sound_data{
			if exit,i:=find(playing, user_list[i]); !exit{
				playing = append(playing, user_list[i])
				go func(){
					for {
						p:=<-pchan
						vc_p.OpusSend <-p.Opus
					}
					playing=append(playing[:i], playing[i+1:]...)
				}()
			}
		}
	}*/
}

func find(slice []uint32, ssrc uint32) (bool, int) {
	for i,e := range slice{
		if e==ssrc{
			return true, i
		}
	}
	return false, len(slice)
}

func opusMixer() (opusData []byte){
	var pmc_list []int16
	for i,p:=range sound_data{pmc_list[i]=decorder_list[i].Decode(p.Opus, 960, false)}
	mixed := make([]int16, len(pmc_list[0]))
	for i := range mixed {
		var v int = 0 
		for _,pcm:=range pmc_list{ v+=int(pcm) }
		v/=len(pmc_list)
		if v > 32767 {
			v = 32767 
		} else if v < -32768 {
			v = -32768
		}
		mixed[i] = int16(v)
	}
	opusData, _ := encoder.Encode(mixed, 960, 960*2)
	return
}