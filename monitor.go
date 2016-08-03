package main

import (
	"flag"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"time"
)

var (
	isLeader bool
)

func must(other string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s:   %s", other, err))
	}
}

func connect(server string) *zk.Conn {
	zks := strings.Split(server, ",")
	conn, _, err := zk.Connect(zks, time.Second)
	must("connect", err)
	return conn
}

func mirror(conn *zk.Conn, path string) (chan []string, chan error) {
	snapshots := make(chan []string)
	errors := make(chan error)
	go func() {
		for {
			snapshot, _, events, err := conn.ChildrenW(path)
			if err != nil {
				errors <- err
				return
			}
			snapshots <- snapshot //必须在evt := <-events
			evt := <-events
			fmt.Println("changing....")
			if evt.Err != nil {
				errors <- evt.Err
				return
			}
		}
	}()
	return snapshots, errors
}

func ChangeToLeader(conn *zk.Conn, masters []string, path string, ip string) {
	if len(masters) == 0 {
		fmt.Println("No leader, I change to leader")
		fmt.Println("start the server")
		fmt.Println("start succeed")
		isLeader = true
		return
	}
	leader := masters[0]
	for _, name := range masters {
		ret := strings.Compare(name, leader)
		if ret < 0 {
			leader = name
		}
	}

	fmt.Println(leader)
	leaderIp, _, err := conn.Get(fmt.Sprintf("%s/%s", path, leader))
	must("GetMaster", err)
	fmt.Printf("leaderIp: %s\n", leaderIp)
	if isLeader {
		fmt.Println("Oh, I am alreay leader!Just a another backup is added or down")
	} else if strings.Compare(ip, string(leaderIp)) == 0 {
		fmt.Println("I get the chance, I change to leader")
		fmt.Println("start the server")
		fmt.Println("start succeed")
		isLeader = true
	} else {
		fmt.Println("I can not get the chance")
	}
}

func init() {
	isLeader = false
}

func main() {

	var server string
	var data string
	flag.StringVar(&server, "zk", "127.0.0.1:2181", "the zookeeper cluster")
	flag.StringVar(&data, "data", "127.0.0.1", "the ip of the node, you have to change it")

	flag.Parse()

	conn := connect(server)
	defer conn.Close()

	flags := int32(3)
	acl := zk.WorldACL(zk.PermAll)

	_, err := conn.Create("/zk_test/bughunter", []byte(data), flags, acl)
	must("Create", err)
	snapshots, errors := mirror(conn, "/zk_test")
	for {
		select {
		case snapshot := <-snapshots:
			fmt.Printf("%+v\n", snapshot)
			ChangeToLeader(conn, snapshot, "/zk_test", data)
		case err := <-errors:
			panic(err)
		}
	}
}
