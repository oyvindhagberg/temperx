package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zserge/hid"
	"log"
	"os"
	"time"
)

var (
	rootCmd = &cobra.Command{
		Use: "temperx",
		Long: "Show temperature and humidity as measured by " +
			"TEMPerHUM/TEMPerX USB devices (ID 413d:2107)",
		Run: func(cmd *cobra.Command, args []string) {
			output()
		},
	}

	home    = os.Getenv("HOME")
	tf      float64
	to      float64
	hf      float64
	ho      float64
	conf    string
	verbose bool
)

func main() {
	rootCmd.Flags().Float64Var(&tf, "tf", 1, "Factor for temperature")
	rootCmd.Flags().Float64Var(&to, "to", 0, "Offset for temperature")
	rootCmd.Flags().Float64Var(&hf, "hf", 1, "Factor for humidity")
	rootCmd.Flags().Float64Var(&ho, "ho", 0, "Offset for humidity")
	rootCmd.Flags().StringVarP(&conf, "conf", "c", home+"/.temperx.toml", "Configuration file")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	viper.BindPFlag("tf", rootCmd.Flags().Lookup("tf"))
	viper.BindPFlag("to", rootCmd.Flags().Lookup("to"))
	viper.BindPFlag("hf", rootCmd.Flags().Lookup("hf"))
	viper.BindPFlag("ho", rootCmd.Flags().Lookup("ho"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func output() {
	if conf != "" {
		if verbose == true {
			fmt.Println("Trying to read configuration from:", conf)
		}
		viper.SetConfigFile(conf)
		viper.ReadInConfig()
	}

	tf := viper.GetFloat64("tf")
	to := viper.GetFloat64("to")
	hf := viper.GetFloat64("hf")
	ho := viper.GetFloat64("ho")
	hid_path := "413d:2107:0000:01"
	cmd_raw := []byte{0x01, 0x80, 0x33, 0x01, 0x00, 0x00, 0x00, 0x00}

	if verbose == true {
		fmt.Println("Using the following factors and offsets:")
		fmt.Println("tf:", tf)
		fmt.Println("to:", to)
		fmt.Println("hf:", hf)
		fmt.Println("ho:", ho)
	}

	var hasErrored bool
	var devNum int
	hid.UsbWalk(func(device hid.Device) {
		info := device.Info()
		id := fmt.Sprintf("%04x:%04x:%04x:%02x", info.Vendor, info.Product, info.Revision, info.Interface)
		if id != hid_path {
			return
		}
		devNum++

		if err := device.Open(); err != nil {
			log.Println("Open error: ", err)
			hasErrored = true
			return
		}

		defer device.Close()

		if _, err := device.Write(cmd_raw, 10*time.Second); err != nil {
			log.Println("Output report write failed:", err)
			hasErrored = true
			return
		}

		if buf, err := device.Read(16, 10*time.Second); err == nil {
			tmp := bytesToValue(buf[2], buf[3], tf, to)
			hum := bytesToValue(buf[4], buf[5], hf, ho)
			fmt.Printf("Device %d Temperature: %v, Humidity: %v\n", devNum, tmp, hum)
			if len(buf)>=14 {
				tmp := bytesToValue(buf[10], buf[11], tf, to)
				hum := bytesToValue(buf[12], buf[13], hf, ho)
				fmt.Printf("Device %d Temperature2: %v, Humidity: %v\n", devNum, tmp, hum)
			}
		} else {
			hasErrored = true
			log.Println("Device read failed:", err)
		}
	})
	if hasErrored {
		os.Exit(1)
	}
}

func bytesToValue(hibyte, lowbyte uint8, factor, offset float64) float64 {
	var word = int16(hibyte)<<8 | int16(lowbyte)
	return float64(word)/100*factor + offset
}
