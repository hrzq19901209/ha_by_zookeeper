package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func read(path string, temp string, domain string, ip string) {
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

func main() {
	var path string
	var domain string
	var ip string
	flag.StringVar(&path, "path", "/etc/dnsmasq.hosts", "the file to read")
	flag.StringVar(&domain, "domain", "www.souhuvideolol.com", "the domain of the server")
	flag.StringVar(&ip, "ip", "127.0.0.1", "the float ip of server")
	flag.Parse()
	read(path, "dnsmasq.hosts", domain, ip)
}
