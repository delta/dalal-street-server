package models

import (
	"github.com/Sirupsen/logrus"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
)

type GameStateType uint8

type MarketState struct {
	IsMarketOpen bool
}

// StockDividendState determines if a company gives dividend
type StockDividendState struct {
	StockID       uint32
	GivesDividend bool
}

type OtpVerifiedState struct {
	IsVerified bool
}

type StockBankruptState struct {
	StockId    uint32
	IsBankrupt bool
}

var gameStateTypes = [...]string{
	"MarketStateUpdate",
	"StockDividendStateUpdate",
	"OtpVerifiedStateUpdate",
	"StockBankruptStateUdpate",
}

const (
	MarketStateUpdate GameStateType = iota
	StockDividendStateUpdate
	OtpVerifiedStateUpdate
	StockBankruptStateUpdate
)

func (gsType GameStateType) String() string {
	return gameStateTypes[gsType]
}

// GameState struct defines different variables of the game
type GameState struct {
	UserID uint32
	GsType GameStateType
	Ms     *MarketState
	Sd     *StockDividendState
	Ov     *OtpVerifiedState
	Sb     *StockBankruptState
}

func (g *GameState) ToProto() *models_pb.GameState {
	pGameState := &models_pb.GameState{
		UserId: g.UserID,
	}

	if g.GsType == MarketStateUpdate {
		pGameState.Type = models_pb.GameStateUpdateType_MarketStateUpdate
		pGameState.MarketState = &models_pb.MarketState{
			IsMarketOpen: g.Ms.IsMarketOpen,
		}
	} else if g.GsType == StockDividendStateUpdate {
		pGameState.Type = models_pb.GameStateUpdateType_StockDividendStateUpdate
		pGameState.StockDividendState = &models_pb.StockDividendState{
			StockId:       g.Sd.StockID,
			GivesDividend: g.Sd.GivesDividend,
		}
	} else if g.GsType == OtpVerifiedStateUpdate {
		pGameState.Type = models_pb.GameStateUpdateType_OtpVerifiedStateUpdate
		pGameState.OtpVerifiedState = &models_pb.OtpVerifiedState{
			IsVerified: g.Ov.IsVerified,
		}
	} else if g.GsType == StockBankruptStateUpdate {
		pGameState.Type = models_pb.GameStateUpdateType_StockBankruptStateUpdate
		pGameState.StockBankruptState = &models_pb.StockBankruptState{
			StockId:    g.Sb.StockId,
			IsBankrupt: g.Sb.IsBankrupt,
		}
	}

	return pGameState
}

// SendGameStateUpadate sends gamestateupdate to multicast stream
func SendGameStateUpadate(g *models_pb.GameState) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "SendGameStateUpadate",
		"userId": g.GetUserId(),
		"type":   g.GetType(),
	})

	l.Infof("Sending gameStateUpdate")

	gamestatestream := datastreamsManager.GetGameStateStream()
	gamestatestream.SendGameStateUpdate(g)

	l.Infof("done")

	return nil
}
