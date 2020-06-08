package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	//"github.com/davecheney/mdns"
	"github.com/brutella/dnssd"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ofauchon/rfskipper-adapter/core"
	"github.com/ofauchon/rfskipper-adapter/decoder"
)

const mqttClientID = "rfs-adapter"

var cnfMqttURL *url.URL
var cnfLogFile string
var cnfIsDaemon bool
var logfile *os.File
var cnfTopicSignalRaw, cnfTopicSignalDecoded string

/* TOOLS ****************************************************/

// doLog write message to log file
func doLog(format string, a ...interface{}) {
	t := time.Now()

	if cnfLogFile != "" && logfile == os.Stdout {
		fmt.Printf("INFO: Creating log file '%s'\n", cnfLogFile)
		tf, err := os.OpenFile(cnfLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("ERROR: Can't open log file '%s' for writing, logging to Stdout\n", cnfLogFile)
			fmt.Printf("ERROR: %s\n", err)
		} else {
			fmt.Printf("INFO: log file '%s' is ready for writing\n", cnfLogFile)
			logfile = tf
		}
	}

	// logfile default is os.Stdout
	fmt.Fprintf(logfile, "%s ", string(t.Format("20060102 150405")))
	fmt.Fprintf(logfile, format, a...)
}

func parseArgs() {
	//mqtt://<user>:<pass>@<server>.cloudmqtt.com:<port>
	var tMqtt string
	flag.StringVar(&tMqtt, "mqtt_url", "mqtt://localhost:1883", "MQTT Broker Address")

	flag.StringVar(&cnfTopicSignalRaw, "topic-signal-raw", "signal_raw", "MQTT Topic for publishing raw signals")
	flag.StringVar(&cnfTopicSignalDecoded, "topic-signal-decoded", "signal_decoded", "MQTT Topic for publishing decoded signals")

	flag.StringVar(&cnfLogFile, "log", "", "Path to log file")
	flag.BoolVar(&cnfIsDaemon, "daemon", false, "Run in background")
	flag.Parse()

	var err error
	cnfMqttURL, err = url.Parse(tMqtt)
	if err != nil {
		log.Fatal(err)
	}

	doLog("Conf : MQTT broker URL : %s\n", cnfMqttURL)
	doLog("Conf : MQTT signalRaw topic name : %s\n", cnfTopicSignalRaw)
	doLog("Conf : MQTT signalDecoded topic name : %s\n", cnfTopicSignalDecoded)

}

/* WEBTHINGS ****************************************************/

/*
// FakeGpioHumiditySensor A humidity sensor which updates its measurement every few seconds.
func FakeGpioHumiditySensor() *webthing.Thing {
	thing := webthing.NewThing(
		"urn:dev:ops:my-humidity-sensor-1234",
		"Power Meter",
		[]string{"MultiLevelSensor"},
		"A web connected humidity sensor")

	level := webthing.NewValue(0.0)
	levelDescription := []byte(`{
        "@type": "LevelProperty",
        "title": "Humidity",
        "type": "number",
        "description": "The current humidity in %",
        "minimum": 0,
        "maximum": 100,
        "unit": "percent",
        "readOnly": true
	}`)
	thing.AddProperty(webthing.NewProperty(
		thing,
		"level",
		level,
		levelDescription))

	go func(level webthing.Value) {
		for {
			time.Sleep(3000 * time.Millisecond)
			newLevel := readFromGPIO()
			fmt.Println("setting new humidity level:", newLevel)
			level.NotifyOfExternalUpdate(newLevel)
		}
	}(level)

	return thing
}
*/

/* MQTT *********************************************************/

func mqttCreateClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	return opts
}

func mqttConnect(clientID string, url *url.URL) mqtt.Client {
	opts := mqttCreateClientOptions(clientID, url)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

// mqttListen subscribe a MQTT topic
func mqttListen(client mqtt.Client, uri *url.URL, topic string, pulseChan chan core.PulseTrain) {
	//	client := mqttConnect(mqttClientID, uri)
	client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		//		fmt.Printf("mqtt sub: [%s] %s\n", msg.Topic(), string(msg.Payload()))

		var p core.PulseTrain
		err := json.Unmarshal(msg.Payload(), &p)
		if err != nil {
			doLog("Can't unmmarshal json!" + err.Error())
		} else {
			//	doLog("mmarshaled: %v\n", p)
			pulseChan <- p
		}

	})
}

// push_mqtt push messages to queues
func mqttPush(client mqtt.Client, ps *list.List) {
	for ps.Len() > 0 {
		// Get First In, and remove it
		e := ps.Front()

		if token := client.Publish("signal_decoded", 0, false, e.Value); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		}
		doLog("Publish '%s' \n", e.Value)

		ps.Remove(e)
	}

}

/* MDNS ****************************************************************/

func dnssd_register() {
	cfg := dnssd.Config{
		Name:   "Sensors",
		Type:   "_webthing._tcp",
		Domain: "local",
		Host:   "psx",
		IPs:    []net.IP{net.ParseIP("192.168.1.26")},
		Port:   12345,
	}
	dnssd.NewService(cfg)
}

/* MAIN ****************************************************************/

func main() {

	logfile = os.Stdout
	parseArgs()
	doLog("Starting RFSkipper adapter \n")

	var decoders = []decoder.Decoder{
		decoder.NewPrologueDecoder(),
		decoder.NewTicPulsesV2Decoder()}

	// MultiThings

	/*
		light := MakeDimmableLight()
		sensor := FakeGpioHumiditySensor()
		multiple := webthing.NewMultipleThings([]*webthing.Thing{light, sensor}, "LightAndTempDevice")
	*/

	// Setup HTTP Server REST WebThings API
	/*
		httpServer := &http.Server{Addr: "0.0.0.0:8888"}
		server := webthing.NewWebThingServer(multiple, httpServer, "")
		log.Fatal(server.Start())
	*/

	// Setup mDNS Server

	/*
		host, _ := os.Hostname()
		host, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
	*/

	//

	// Inter routines communicatin
	pulseChan := make(chan core.PulseTrain)

	// Connect MQTT
	client := mqttConnect(mqttClientID, cnfMqttURL)
	go mqttListen(client, cnfMqttURL, cnfTopicSignalRaw, pulseChan)

	ps := list.New()

	// Loop until someting happens
	for {
		p := <-pulseChan
		doLog("New Signal : Length: %d\n", len(p.Pulses))
		doLog("Payload : %v\n", p)

		for _, d := range decoders {
			_, dec := d.Decode(p)
			fmt.Println(d.GetDecoderName(), " ", dec)
		}

		//ps.PushBack(message)
		mqttPush(client, ps)

	}

}
