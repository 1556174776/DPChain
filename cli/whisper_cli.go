package cli

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"p2p-go/common"
	"p2p-go/crypto"
	"p2p-go/discover"
	"p2p-go/logger"
	"p2p-go/nat"
	"p2p-go/server"
	"p2p-go/utils"
	"p2p-go/whisper"
	"runtime"
	"strings"
)

var (
	workpath    = common.GetExecutePath()
	os_seprator = string(os.PathSeparator)
)

var (
	CLI_VERSION         = "0.1"
	WHISPER_COMMANDLINE = "WPC"
	WHISPER_SERVER      = "WPS"
	KEY_DIR             = workpath + os_seprator + "keys"
	KEY_FILE            = KEY_DIR + os_seprator + "local.key"
	PUBKEY_DIR          = workpath + os_seprator + "pubkeys"
	SELFPUBKEY_FILE     = PUBKEY_DIR + os_seprator + "self.pubkey"
	LOG_DIR             = workpath + os_seprator + "log"
	LOG_FILE            = LOG_DIR + os_seprator + "wpc.log"
	NODE_DIR            = workpath + os_seprator + "bootnodes"
	NODE_FILE           = NODE_DIR + os_seprator + "nodes.txt"
	SELF_FILE           = NODE_DIR + os_seprator + "self.txt"
	LOCAL_ADDRESS       = "127.0.0.1:30300"
	ACTION_DIR          = workpath + os_seprator + "action"
	ACTION_FILE         = ACTION_DIR + os_seprator + "actionlist.txt"
)

var WPS_LOGGER, WPC_LOGGER *logger.Logger

type CommandLine struct {
	registerPubKey map[string]ecdsa.PublicKey
	registerPrvKey map[string]ecdsa.PrivateKey
}

func NewCommandLine() *CommandLine {
	cmd := CommandLine{}
	cmd.registerPubKey = make(map[string]ecdsa.PublicKey)
	cmd.registerPrvKey = make(map[string]ecdsa.PrivateKey)
	return &cmd
}

func (c *CommandLine) startServer(servername string, discoverFlag bool, maxPeer int, boostNodes []*discover.Node, pro server.Protocol, addr string, key *ecdsa.PrivateKey) *server.Server {
	name := common.MakeName(servername, CLI_VERSION)
	var wps server.Server
	if !discoverFlag {
		wps = server.Server{
			PrivateKey: key,
			MaxPeers:   maxPeer,
			Name:       name,
			Protocols:  []server.Protocol{pro},
			ListenAddr: addr,
			NAT:        nat.Any(),
		}
	} else {
		wps = server.Server{
			PrivateKey:     key,
			MaxPeers:       maxPeer,
			Discovery:      true,
			BootstrapNodes: boostNodes,
			Name:           name,
			Protocols:      []server.Protocol{pro},
			NAT:            nat.Any(),
			ListenAddr:     addr,
		}
	}
	WPS_LOGGER.Infof("start p2p server\n")
	if err := wps.Start(); err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
	}
	return &wps
}

func (c *CommandLine) Run() {

	fmt.Println("This is a tiny whisper client ot test the P2P module.")
	fmt.Println("Version: ", CLI_VERSION)

	logfile, _ := os.OpenFile(LOG_FILE, os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer logfile.Close()

	logger.AddLogSystem(logger.NewStdLogSystem(logfile, log.LstdFlags, logger.InfoLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(logfile, log.LstdFlags, logger.WarnLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.WarnLevel))
	WPC_LOGGER = logger.NewLogger(WHISPER_COMMANDLINE)
	WPS_LOGGER = logger.NewLogger(WHISPER_SERVER)
	WPC_LOGGER.Infoln("whisper client commandline start")
	logger.Flush()

	if err := utils.DirCheck(KEY_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(LOG_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(NODE_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(ACTION_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	if err := utils.DirCheck(PUBKEY_DIR); err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}

	clientStartCmd := flag.NewFlagSet("clientstart", flag.ExitOnError)
	clientName := clientStartCmd.String("name", "whisper-client", "the name of the whisper client")
	keyPath := clientStartCmd.String("keypath", KEY_FILE, "the file path of the node's private key, if nil will try to find key in the default path")
	maxPeer := clientStartCmd.Int("peernum", 10, "the max number of peers to connect")
	useDiscovery := clientStartCmd.Bool("discovery", false, "whether to use discovery")
	listenAddr := clientStartCmd.String("addr", LOCAL_ADDRESS, "the listen address of the whisper client")
	nodePath := clientStartCmd.String("nodepath", NODE_FILE, "the file path of nodes used to bootstrap, if nil will try the default path")
	autoMode := clientStartCmd.Bool("automode", false, "if automode, wpc will do actions in action list autonomously")
	actionPath := clientStartCmd.String("actionpath", ACTION_FILE, "used in automode, which stores the actions")

	keyGenerateCmd := flag.NewFlagSet("keygenerate", flag.ExitOnError)
	keyStorePath := keyGenerateCmd.String("keypath", KEY_FILE, "the file path of the generated key, if nil will use default path")

	urlSaveCmd := flag.NewFlagSet("urlsave", flag.ExitOnError)
	urlAddr := urlSaveCmd.String("addr", LOCAL_ADDRESS, "the listen address of the whisper client")
	urlSavePath := urlSaveCmd.String("savepath", SELF_FILE, "the file path to save url, if nil will use default path")
	urlKeyPath := urlSaveCmd.String("keypath", KEY_FILE, "the file path of the node's private key, if nil will try to find key in the default path")

	printUsage := func() {
		fmt.Println("Usage is as follows:")
		fmt.Println("clientstart")
		clientStartCmd.PrintDefaults()
		fmt.Println("keygenerate")
		keyGenerateCmd.PrintDefaults()
		fmt.Println("urlsave")
		urlSaveCmd.PrintDefaults()
	}

	if len(os.Args) == 1 {
		printUsage()
	} else {
		switch os.Args[1] {
		case "clientstart":
			WPC_LOGGER.Infoln("parse command: clientstart")
			err := clientStartCmd.Parse(os.Args[2:])
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}
		case "keygenerate":
			WPC_LOGGER.Infoln("parse command: keygenerate")
			err := keyGenerateCmd.Parse(os.Args[2:])
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}
		case "urlsave":
			WPC_LOGGER.Infoln("parse command: urlsave")
			err := urlSaveCmd.Parse(os.Args[2:])
			if err != nil {
				WPC_LOGGER.Infof("error: %v.\n", err)
				runtime.Goexit()
			}

		default:
			printUsage()
		}
	}

	if clientStartCmd.Parsed() {
		WPC_LOGGER.Infof("start whisper client")
		shh := whisper.New()
		defer shh.Stop()
		selfKey, err := utils.LoadKey(*keyPath)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
			runtime.Goexit()
		}
		bootNodes, err := utils.ReadNodesUrl(*nodePath)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
			runtime.Goexit()
		}
		wps := c.startServer(*clientName, *useDiscovery, *maxPeer, bootNodes, shh.Protocol(), *listenAddr, &selfKey)
		defer wps.Stop()

		if *autoMode {
			WPC_LOGGER.Infof("auto mode start")
			err := c.autoAct(*actionPath, wps, shh)
			if err != nil {
				WPS_LOGGER.Infof("error: %v.\n", err)
			}
			WPC_LOGGER.Infof("auto mode end")
		} else {
			WPC_LOGGER.Infof("action interpreter start")
			c.actionInterpreter(wps, shh)
			WPC_LOGGER.Infof("action interpreter end")
		}
	}

	if keyGenerateCmd.Parsed() {
		WPC_LOGGER.Infof("generate and save key")
		c.createKey(*keyStorePath)
	}

	if urlSaveCmd.Parsed() {
		WPC_LOGGER.Infof("generate URL and save")
		c.saveUrl(*urlAddr, *urlKeyPath, *urlSavePath)
	}

	WPC_LOGGER.Infoln("whisper client commandline stop")
	logger.Flush()
	runtime.Goexit()
}

func (c *CommandLine) createKey(path string) {
	key, err := crypto.GenerateKey()
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	err = utils.SaveKey(*key, path)
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
	err = utils.SaveSelfPublicKey(key.PublicKey, SELFPUBKEY_FILE)
	if err != nil {
		WPC_LOGGER.Infof("error: %v.\n", err)
	}
}

func (c *CommandLine) autoAct(actionList string, wps *server.Server, shh *whisper.Whisper) error {
	if !common.FileExist(actionList) {
		return errors.New("error in autoAct: no such action list")
	}
	fi, err := os.Open(actionList)
	if err != nil {
		return err
	}
	defer fi.Close()

	br := bufio.NewReader(fi)
	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		err = c.doAction(string(line), wps, shh)
		if err != nil {
			return err
		}
	}
	return nil

}

func (c *CommandLine) actionInterpreter(wps *server.Server, shh *whisper.Whisper) {
	printActionUsage()
	reader := bufio.NewReader(os.Stdin)
scanline:
	for {
		logger.Flush()
		var line string
		fmt.Println("Please input action:")
		line, _ = reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "Exit" || line == "exit" || line == "EXIT" || line == "quit" || line == "Quit" || line == "QUIT" || line == "q" || line == "Q" {
			break scanline
		}
		err := c.doAction(line, wps, shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	}

}

func (c *CommandLine) saveUrl(listenaddr, keypath, savepath string) {
	selfKey, err := utils.LoadKey(keypath)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
	parsedNode, err := discover.ParseUDP(&selfKey, listenaddr)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
	url := parsedNode.String()
	err = utils.SaveSelfUrl(url, savepath)
	if err != nil {
		WPS_LOGGER.Infof("error: %v.\n", err)
		runtime.Goexit()
	}
}
