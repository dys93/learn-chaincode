package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var FarmWeatherIndexStr = "_farmindex"    //name for the key/value that will store a list of all known marbles
var ActiveInsuranceStr = "_openinsurance" //name for the key/value that will store all open trades
var UserIndexStr = "_userindex"

type User struct {
	Name string `json:"name"`
	Coin int    `json:"Coin"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	var empty []string
	jsonAsBytes, _ := json.Marshal(empty) //marshal an emtpy array of strings to clear the index
	err = stub.PutState(FarmWeatherIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	err = stub.PutState(UserIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	var insurances ActiveInsurance
	jsonAsBytes, _ = json.Marshal(insurances) //clear the open trade struct
	err = stub.PutState(ActiveInsuranceStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ============================================================================================================================
// Run - Our entry point for Invocations - [LEGACY] obc-peer 4/25/2016
// ============================================================================================================================
func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "write" { //writes a value to the chaincode state
		return t.Write(stub, args)
	} else if function == "create_user" { //create a new marble
		return t.create_user(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation")
}

// ============================================================================================================================
// Query - Our entry point for Queries
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query")
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name) //get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil //send it onward
}

// ============================================================================================================================
// Write - write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) Write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Create User - create a new User,
// ============================================================================================================================

func (t *SimpleChaincode) create_user(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3
	//  'name'   'money'
	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	//input sanitation
	fmt.Println("- start create user")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}

	name := strings.ToLower(args[0])
	coin, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("2rd argument must be a numeric string")
	}

	//check if marble already exists
	UserAsBytes, err := stub.GetState(name)
	if err != nil {
		return nil, errors.New("Failed to get user name")
	}
	res := User{}
	json.Unmarshal(UserAsBytes, &res)
	if res.Name == name {
		fmt.Println("This user arleady exists: " + name)
		fmt.Println(res)
		return nil, errors.New("This user arleady exists") //all stop a user by this name exists
	}

	//build the user json string manually
	var user User
	user.Name = name
	user.Coin = coin
	UserAsBytes, err = json.Marshal(user)
	err = stub.PutState(name, UserAsBytes) //store marble with id as key
	if err != nil {
		return nil, err
	}

	//get the user index
	UsersAsBytes, err := stub.GetState(UserIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get user index")
	}
	var UserIndex []string
	json.Unmarshal(UsersAsBytes, &UserIndex) //un stringify it aka JSON.parse()

	//append
	UserIndex = append(UserIndex, name) //add user name to index list
	fmt.Println("! User index: ", UserIndex)
	jsonAsBytes, _ := json.Marshal(UserIndex)
	err = stub.PutState(UserIndexStr, jsonAsBytes) //store name of user

	fmt.Println("- end create User")
	return nil, nil
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func (t *SimpleChaincode) update_user_account(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	//   0       1       2     3
	//  'userName'   'coin_change' 
	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}

	//input sanitation
	fmt.Println("- start create user")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	var userName string
	userName = args[0]
	coinChange, err := strconv.Atoi(args[1])
	if err != nil {
		msg := "is not a numeric string " + args[1]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}
	
	userAsByte, err := stub.GetState(username)
	if err != nil {
		return nil, errors.New("user don't exist")
	}
	var lucky_dog User
	json.Unmarshal(userAsByte, &lucky_dog)
	lucky_dog.Coin = lucky_dog.Coin + coinChange
	if lucky_dog.Coin + coinChange < 0{
		return nil, errors.New("User don't have enough coin")
	}
	userAsByte, err = json.Marshal(lucky_dog)
	stub.PutState(username, userAsByte)

	return nil, nil
}
