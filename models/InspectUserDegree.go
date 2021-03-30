package models

import (

	"sort"
	"github.com/sirupsen/logrus"

)

//Result of inspect users
type InspectDegreeDetails struct {
	Volume   map[int32]int64
	Position map[int32]int32
}

//Result of number of users
type NumberOfUsersResult struct{
	Number int32
}

//Transaction results
type TransactionGraph struct{
	Fromid int32
	Toid int32
	Volume int64
}

//Struct for sorting volume
type VolumeSort struct{
	volume int64
	id int32
}

//Struct for sorting volume
type VolumeSortArray struct{
	elements []VolumeSort
}

//Struct for getting user number to userID mapping
type UserId struct{
	Id int32
}

//Function to get number of users from db
func getNumberOfUsers() int32{

	l := logger.WithFields(logrus.Fields{
		"method":  "getNUmberOfUsers",
	})

	l.Debugf("Attempting to get total users")

	var numRes NumberOfUsersResult

	db := getDB()

	err := db.Raw("SELECT count(*) as number from Users").Scan(&numRes).Error

	if err != nil{
		l.Errorf("Error getting total number of users")
	}
	
	return numRes.Number

}

//Funcion to build graph and volume of transaction of each user
//with nodes of degree 1
func buildGraph(nnodes int32)(res1 InspectDegreeDetails){

	l := logger.WithFields(logrus.Fields{
		"method":  "buildGraph",
	})

	//Adjacency matrix storing weight of each edge 
	var weights[2001][2001] int64

	var transDetails[] TransactionGraph

	db := getDB()

    //Query to get transactions made between users
	//While making edges cash flows between bidding user and Asking user
	//This determines the direction of edge in the graph
	err := db.Raw("SELECT b.userId as fromid, a.userId as toid, t.total as volume FROM OrderFills o, Transactions t, Asks a, Bids b WHERE o.transactionId = t.id AND o.bidId = b.id AND o.askId = a.id").Scan(&transDetails).Error

	var users[] UserId

	//Get user ids 
	//This is necessary as in db user ids are not sequential and hence a mapping
	//is required between user number and user id
	err = db.Raw("SELECT id from Users").Scan(&users).Error

	//For userId to number mapping
	var userMap map[int32]int32

	userMap = make(map[int32]int32)

	//Map from userId -> user number
	for i := 0;i < len(users);i++{
		userMap[users[i].Id] = int32(i+1)
	}

	//Update the weights using transactions with help of user id map
	for i := 0;i < len(transDetails);i++{
		weights[userMap[transDetails[i].Fromid]][userMap[transDetails[i].Toid] ]+= transDetails[i].Volume
	}

	//First all nodes which are of degree one
	//are found and the state is updated in the boolean array
	var isDegreeOne[2001] bool
	var i, j int32

	//Direction of transaction is immaterial so both directions are summed up
	for i = 1; i <= nnodes;i++{
		for j = 1;j <= nnodes;j++{
			if i > j{
				weights[i][j] += weights[j][i]
				weights[j][i] = weights[i][j]
			}
		}
	}

	//Find degree one nodes
	for i = 1;i <= nnodes;i++{
		
		count := 0

		for j = 1;j <= nnodes;j++{
			if weights[i][j] > 0{
				count += 1
			}
		}
		if count == 1{
			isDegreeOne[i] = true
		}
	}

	var res InspectDegreeDetails

	var volumeVals VolumeSortArray

	//Find volume of transaction from every node to degree one nodes
	for i = 1;i <= nnodes;i++{
		var temp int64 = 0
		for j = 1;j <= nnodes;j++{
			if isDegreeOne[j] {
				temp += weights[i][j]
			}
		}
		var curVol VolumeSort

		curVol.volume = temp
		curVol.id = users[i - 1].Id
		volumeVals.elements = append(volumeVals.elements, curVol)
	}

	//Sort by volume in descending order
	sort.Slice(volumeVals.elements, func(i, j int) bool {
		return volumeVals.elements[i].volume > volumeVals.elements[j].volume
	})

	res.Volume = make(map[int32]int64)
	res.Position = make(map[int32]int32)
	
	for i = 0;i < int32(len(volumeVals.elements));i++{
		res.Volume[volumeVals.elements[i].id] = volumeVals.elements[i].volume
		res.Position[volumeVals.elements[i].id] = i+1
	}


	if err != nil{
		l.Errorf("Error getting user graph")
	}

	return res
}

func InspectUserDegree() (InspectDegreeDetails, error) {

	var inspectUserEntries InspectDegreeDetails
	numUsers := getNumberOfUsers()

	inspectUserEntries = buildGraph(numUsers)
	return inspectUserEntries, nil
}
