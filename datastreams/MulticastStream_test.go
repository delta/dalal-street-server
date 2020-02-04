package datastreams

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func GetMockMulticastStream(t *testing.T) (*gomock.Controller, *multicastStream, *listener, string, string) {
	var sessionID string = "123456789"
	var groupID string = "1234"
	var Groups = &groupsMap{m: make(map[uint32]BroadcastStream)}

	var Update <-chan struct{}
	var Done chan string
	mockcontrol := gomock.NewController(t)

	var mocklistener listener = &listener{
		update: Update,
		done:   Done,
	}

	mockbroadcastStream := mocks.NewBroadcastStream(mockcontrol)

	multicaststream := &multicastStream{
		groups: Groups,
	}

	return nil, mockcontrol, multicaststream, mocklistener, sessionID, groupID

}
func TestMulticastAddListener(t *testing.T) {

	mockcontrol, multicaststream, mlistener, sessioID, groupID := GetMockMulticastStream(t)

	multicaststream.groups.Lock()
	group, exists := multicaststream.groups.m[groupId]

	multicaststream.groups.m[groupId] = NewBroadcastStream()
	group = multicaststream.groups.m[groupID]
	multicaststream.groups.Unlock()

}
func TestMulticastRemoveListener(t *testing.T) {}
func TestBroadcastUpdateToGroup(t *testing.T)  {}
func TestMakeGlobalBroadcast(t *testing.T)     {}
