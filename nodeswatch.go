package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	currentIp string
	server    string //zk
	path      string //dnsmasq.hosts
	domain    string //domain
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

func getLeader(conn *zk.Conn, masters []string, path string) {
	if len(masters) == 0 {
		fmt.Println("No leader")
		fmt.Println("I have to tell dns no leader")
		currentIp = ""
		updateDNS(currentIp)
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
	must("GetLeader", err)
	fmt.Printf("leaderIp: %s\n", leaderIp)
	if strings.Compare(string(leaderIp), currentIp) == 0 {
		fmt.Println("Oh, The leader does not change, I dont need to tell the dns")
	} else {
		fmt.Println("Oh, The leader change, I have to tell the dns")
		currentIp = string(leaderIp)
		updateDNS(currentIp)
	}
}

func updateDNS(ip string) {
	temp := "dnsmasq.hosts"
	fi, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	os.Remove(temp)
	wi, err := os.Create(temp)
	if err != nil {
		panic(err)
	}
	defer wi.Close()

	r := bufio.NewReader(fi)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		if !strings.Contains(line, domain) {
			io.WriteString(wi, line)
		}
	}
	if ip != "" {
		newDomain := fmt.Sprintf("%s %s\n", ip, domain)
		io.WriteString(wi, newDomain)
	}

	wi.Close()
	wi, err = os.Open(temp)
	if err != nil {
		panic(err)
	}
	defer wi.Close()

	os.Remove(path)
	fni, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer fni.Close()

	r = bufio.NewReader(wi)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		io.WriteString(fni, line)
	}
	cmd := exec.Command("/bin/sh", "-c", "systemctl restart dnsmasq")
	err = cmd.Run()
	if err != nil {
		panic(err.Error())
	}
}

func init() {
	currentIp = ""
	flag.StringVar(&server, "zk", "127.0.0.1:2181", "the zookeeper cluster")
	flag.StringVar(&path, "path", "/etc/dnsmasq.hosts", "the path of dnsmasq hosts")
	flag.StringVar(&domain, "domain", "www.sohucloud.com", "the domain of the server")
	flag.Parse()
}

func main() {

	conn := connect(server)
	defer conn.Close()

	snapshots, errors := mirror(conn, "/zk_test")
	for {
		select {
		case snapshot := <-snapshots:
			fmt.Printf("%+v\n", snapshot)
			getLeader(conn, snapshot, "/zk_test")
		case err := <-errors:
			panic(err)
		}
	}
}
