package models

import (

	"fmt"

	"github.com/sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"

)


func (c *InspectComponentResult) ToProto() *models_pb.Cluster {
	pCluster:= &models_pb.Cluster{
		Members:   c.members,
		Volume: c.volume,
	}
	return pCluster
}

type Stack []int32

func (s Stack) Push(v int32) Stack {
    return append(s, v)
}

func (s Stack) Pop() (Stack, int32) {

	if len(s) == 0{
		fmt.Println("Stack empty")
	}
    l := len(s)
    return  s[:l-1], s[l-1]
}

//Inspect Component Result
type InspectComponentResult struct{
	members []int32
	volume int64
}

type AdjacencyList struct{
	nodes int32
	edges map[int32][]int32
}

//Funcion to build graph
func getComponents(nnodes int32)(res1[] InspectComponentResult){

	l := logger.WithFields(logrus.Fields{
		"method":  "buildGraph",
	})

	var weights[2001][2001] int64

	var transDetails[] TransactionGraph

	var res[] InspectComponentResult

	db := getDB()

	err := db.Raw("SELECT b.userId as fromid, a.userId as toid, t.total as volume FROM OrderFills o, Transactions t, Asks a, Bids b WHERE o.transactionId = t.id AND o.bidId = b.id AND o.askId = a.id").Scan(&transDetails).Error

	var users[] UserId

	//Get user ids
	err = db.Raw("SELECT id from Users").Scan(&users).Error

	//For userId to number mapping
	var userMap map[int32]int32

	userMap = make(map[int32]int32)

	for i := 0;i < len(users);i++{
		userMap[users[i].Id] = int32(i+1)
	}

	for i := 0;i < len(transDetails);i++{
		weights[userMap[transDetails[i].Fromid]][userMap[transDetails[i].Toid]] += transDetails[i].Volume
	}

	var listGraph AdjacencyList
	var reversedGraph AdjacencyList

	var i, j, k int32

	listGraph.nodes = nnodes
	reversedGraph.nodes = nnodes

	listGraph.edges = make(map[int32][]int32)
	reversedGraph.edges = make(map[int32][]int32)

	for i = 1;i <= nnodes;i++{
		for j = 1;j <= nnodes;j++{
			if weights[i][j] > 0{
				listGraph.edges[i] = append(listGraph.edges[i], j)
				reversedGraph.edges[j] = append(listGraph.edges[j], i)
			}
		}
	}

	if err != nil{
		l.Errorf("Error")
	}

	var nodeStack Stack

	var visited[] bool

	for i = 1;i <= nnodes + 1;i++{
		visited = append(visited, false)
	}


	var order[] int32
	var top int32

	for i = 1;i <= nnodes;i++{
		if !visited[i]{

			nodeStack = nodeStack.Push(i)
			
			for len(nodeStack) > 0 {
				
				nodeStack, top = nodeStack.Pop()

				if !visited[top]{
					visited[top] = true
					

					for j = 0;j < int32(len(listGraph.edges[top]));j++{
						if !visited[listGraph.edges[top][j]]{
							nodeStack = nodeStack.Push(listGraph.edges[top][j])
						}
					}
				}
				order = append(order, top)
			}
		
		}
	}

	for i = 1;i <= nnodes;i++{
		visited[i] = false
	}

	for i = 1;i <= nnodes;i++{
		if !visited[order[i - 1]]{
			var cur[] int32
			nodeStack = nodeStack.Push(order[i - 1])
			
			for len(nodeStack) > 0 {
				
				nodeStack, top = nodeStack.Pop()

				if !visited[top]{
					visited[top] = true
					cur = append(cur, users[top - 1].Id)
			
					for j = 0;j < int32(len(reversedGraph.edges[top]));j++{
						if !visited[reversedGraph.edges[top][j]]{
							nodeStack = nodeStack.Push(reversedGraph.edges[top][j])
						}
					}
				}
			}

			var curComponent InspectComponentResult
			curComponent.volume = 0
			curComponent.members = cur
			res = append(res, curComponent)
		}
	}

	for i = 0;i < int32(len(res));i++{

		for j = 0;j < int32(len(res[i].members));j++{
			for k = 0;k < int32(len(res[i].members));k++{
				res[i].volume += weights[userMap[res[i].members[j]]][userMap[res[i].members[k]]]
			}
		} 
	}

	return res
}

func InspectComponents() (finalRes[] InspectComponentResult,e error) {

	numUsers := getNumberOfUsers()

	curRes := getComponents(numUsers)
	return curRes, nil
}