package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Variables used for command line parameters
var (
	Token string
	APIUrl string
	conn *grpc.ClientConn
	beaconClient eth.BeaconChainClient
	nodeClient eth.NodeClient
	log = logrus.WithField("prefix", "prysmBot")
)

func init() {
	flag.StringVar(&Token, "token", "", "Bot Token")
	flag.StringVar(&APIUrl, "api-url", "", "API Url for gRPC")
	flag.Parse()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	conn, err = grpc.Dial(APIUrl, grpc.WithInsecure())
	if err != nil {
		log.Error ("Failed to dial: %v", err)
	}
	beaconClient = eth.NewBeaconChainClient(conn)
	nodeClient = eth.NewNodeClient(conn)
	defer conn.Close()

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !whitelistedChannel(m.ChannelID)  {
		return
	}
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore all messages that don't start with "!".
	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	fullCommand := m.Content[1:]
	// If the message is "ping" reply with "Pong!"
	if fullCommand == "ping" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Pong!")
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}
	if fullCommand == "help" && helpOkayChannel(m.ChannelID) {
		embed := fullHelpEmbed()
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}

	if isRandomCommand(fullCommand) {
		result := getRandomResult(fullCommand)
		_, err := s.ChannelMessageSend(m.ChannelID, result)
		if err != nil {
			log.WithError(err).Errorf("Error handling command %s", fullCommand)
			return
		}
	}

	splitCommand := strings.Split(fullCommand, ".")
	if fullCommand == splitCommand[0] {
		return
	} else if len(splitCommand) > 1 && strings.TrimSpace(splitCommand[1]) == "" {
		return
	}
	commandGroup := splitCommand[0]
	endOfCommand := strings.Index(splitCommand[1], " ")
	var parameters []string
	if endOfCommand == -1 {
		endOfCommand = len(splitCommand[1])
	} else {
		parameters = strings.Split(splitCommand[1][endOfCommand:], ",")
		for i, param := range parameters {
			parameters[i] = strings.TrimSpace(param)
		}
	}
	command := splitCommand[1][:endOfCommand]

	var cmdFound bool
	var cmdGroupFound bool
	var reqGroup *botCommandGroup
	for _, flagGroup := range allFlagGroups {
		if flagGroup.name == commandGroup || flagGroup.shorthand == commandGroup {
			cmdGroupFound = true
			reqGroup = flagGroup
			for _, cmd := range reqGroup.commands {
				if command == cmd.command || command == cmd.shorthand || command == "help"{
					cmdFound = true
				}
			}
		}
	}
	if !cmdGroupFound || !cmdFound {
		return
	}

	if command == "help" && helpOkayChannel(m.ChannelID) {
		embed := specificHelpEmbed(reqGroup)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}

	var result string
	switch commandGroup {
	case currentCommandGroup.name, currentCommandGroup.shorthand:
		result = getHeadCommandResult(command)
	case stateCommandGroup.name, stateCommandGroup.shorthand:
		result = getStateCommandResult(command, parameters)
	case valCommandGroup.name, valCommandGroup.shorthand:
		result = getValidatorCommandResult(command, parameters)
	case blockCommandGroup.name, blockCommandGroup.shorthand:
		result = getBlockCommandResult(command, parameters)
	default:
		result = "Command not found, sorry!"
	}
	if result == "" {
		return
	}
	_, err := s.ChannelMessageSend(m.ChannelID, result)
	if err != nil {
		log.WithError(err).Errorf("Error handling command %s", fullCommand)
		return
	}
}

func helpOkayChannel(channelID string) bool {
	switch channelID {
	case "691473296696410164":
		return true
	case "701148358445760573":
		return true
	case "696886109589995521": // #random in Prsym Discord.
		return true
	default:
		return false
	}
}

func whitelistedChannel(channelID string) bool {
	switch channelID {
	case "476588476393848832": // #general in Prsym Discord.
		return true
	default:
		return helpOkayChannel(channelID)
	}
}