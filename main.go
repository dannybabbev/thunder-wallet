package main

import (
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"net/http"
	"os"
)

var lnClient lnrpc.LightningClient
var macaroon string

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type configuration struct {
	nodeAddress string
	port        string
}

func main() {
	logrus.Debug("starting wallet")
	var conf configuration
	if addr := os.Getenv("NODE"); addr == "" {
		conf.nodeAddress = "206.189.33.20:10009"
		logrus.WithField("node-address", conf.nodeAddress).Debug("setting default node address")
	}

	if port := os.Getenv("PORT"); port == "" {
		conf.port = "8080"
		logrus.WithField("port", conf.port).Debug("setting default port")
	}

	creds, err := credentials.NewClientTLSFromFile("secret/tls.cert", "")
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create credentials")
	}
	lnConn, err := grpc.Dial(
		conf.nodeAddress,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not dial lnd gRPC")
	}

	mac, err := ioutil.ReadFile("secret/data/chain/bitcoin/testnet/admin.macaroon")
	if err != nil {
		logrus.WithField("err", err).Fatal("could not read macaroon")
	}
	macaroon = hex.EncodeToString(mac)
	lnClient = lnrpc.NewLightningClient(lnConn)

	info, err := lnClient.GetInfo(getStdContext(), &lnrpc.GetInfoRequest{})
	if err != nil {
		logrus.WithField("err", err).Fatal("Could not get info from lnd. Is your configuration correct?")
	}
	logrus.WithField("info", info).Debug("lnd info")

	r := gin.New()
	r.Use(gin.Recovery())
	r.Static("/assets", "./public/assets")
	r.LoadHTMLGlob("./public/templates/*")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	api := r.Group("/api")
	api.GET("/info", func(c *gin.Context) {
		info, err := lnClient.GetInfo(getStdContext(), &lnrpc.GetInfoRequest{})
		if err != nil {
			logrus.WithField("err", err).Error("could not get info from lnd")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pubkey":        info.IdentityPubkey,
			"alias":         info.Alias,
			"bestBlockHash": info.BlockHash,
			"uri":           info.Uris,
			"version":       info.Version,
		})
	})

	r.Run(":" + conf.port)
}

func getStdContext() context.Context {
	meta := metadata.Pairs("macaroon", macaroon)
	return metadata.NewOutgoingContext(context.Background(), meta)
}
