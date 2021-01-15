package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const (
	baseUrl     = "https://www.autotrader.co.uk/car-search?postcode=E144AD&make=FORD&model=FOCUS&price-to=25000&include-delivery-option=on&body-type=Hatchback&transmission=Manual&year-from=2015&onesearchad=Used,Nearly%20New,New&advertising-location=at_cars&page="
	outputFile  = "car_info.json"
	maxNumPages = 100
)

func main() {
	cars := getAllCars()
	fmt.Printf("parsed info on %d cars\n", len(cars))

	err := writeOutput(cars)
	noErr(err)
}

type CarInfo struct {
	Price      uint `json:"price"` // in pence
	Year       uint `json:"year"`
	Mileage    uint `json:"mileage"`
	EngineSize uint `json:"engine_size"` // CC
}

func getAllCars() []*CarInfo {
	var allCars []*CarInfo

	for i := uint64(0); i < maxNumPages; i++ {
		cars, err := getPage(i)
		if err != nil {
			fmt.Printf("error getting page %d: %s\n", i, err.Error())
		}
		allCars = append(allCars, cars...)
	}

	return allCars
}

func writeOutput(cars []*CarInfo) error {
	data, err := json.Marshal(cars)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputFile, data, 0644)
}

func getPage(pageNum uint64) ([]*CarInfo, error) {
	url := baseUrl + strconv.FormatUint(pageNum, 10)
	rsp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "while making HTTP request to %s", url)
	}

	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status %s", rsp.Status)
	}

	doc, err := html.Parse(rsp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing as HTML")
	}

	var cars []*CarInfo

	var parseHTMLNode func(n *html.Node)
	parseHTMLNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for i := range n.Attr {
				if n.Attr[i].Key == "class" && n.Attr[i].Val == "product-card-content" {
					carInfo, err := parseCarNode(n)
					if err != nil {
						fmt.Println("error parsing car node:", err.Error())
						return
					}
					cars = append(cars, carInfo)
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseHTMLNode(c)
		}
	}
	parseHTMLNode(doc)

	return cars, nil
}

var priceRe = regexp.MustCompile(`^£([\d,]+)$`)

func parseCarNode(n *html.Node) (*CarInfo, error) {
	price, err := parseCarPrice(n)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing car price")
	}

	carInfo, err := parseCarSpecs(n)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing car specs")
	}

	carInfo.Price = price
	return carInfo, nil
}

var errPriceNotFound = errors.New("could not parse a price under this node")

func parseCarPrice(n *html.Node) (uint, error) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			for i := range c.Attr {
				if c.Attr[i].Key == "class" && c.Attr[i].Val == "product-card-pricing__price" {
					for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
						if cc.Type == html.ElementNode && cc.Data == "span" {
							rawPrice := cc.FirstChild.Data
							matches := priceRe.FindStringSubmatch(rawPrice)
							if len(matches) != 2 {
								return 0, errors.Errorf("cannot parse price '%s'", rawPrice)
							}

							replaced := strings.ReplaceAll(matches[1], ",", "")
							val, err := strconv.ParseUint(replaced, 10, 32)
							if err != nil {
								return 0, errors.Wrapf(err, "while parsing %s as a uint", replaced)
							}

							return uint(val), nil
						}
					}
				}
			}
		}

		val, err := parseCarPrice(c)
		if err == nil {
			return val, nil
		}
		if err != errPriceNotFound {
			return 0, err
		}
	}

	return 0, errPriceNotFound
}

func parseCarSpecs(n *html.Node) (*CarInfo, error) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "ul" {
			for i := range c.Attr {
				if c.Attr[i].Key == "class" && c.Attr[i].Val == "listing-key-specs" {
					i := 0
					info := new(CarInfo)

					for c := c.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode && c.Data == "li" {
							cc := c.FirstChild
							if cc == nil || cc.Type != html.TextNode {
								return nil, errors.New("unexpected li node")
							}

							switch i {
							case 0:
								year, err := parseYear(cc.Data)
								if err != nil {
									return nil, errors.Wrap(err, "while parsing year")
								}
								info.Year = year

							case 2:
								mileage, err := parseMileage(cc.Data)
								if err != nil {
									return nil, errors.Wrap(err, "while parsing mileage")
								}
								info.Mileage = mileage

							case 3:
								engineSize, err := parseEngineSize(cc.Data)
								if err != nil {
									return nil, errors.Wrap(err, "while parsing engine size")
								}
								info.EngineSize = engineSize

							default:
								if i > 3 {
									return info, nil
								}
							}

							i += 1
						}
					}
				}
			}
		}

		info, err := parseCarSpecs(c)
		if err == nil {
			return info, nil
		}
		if err != errPriceNotFound {
			return nil, err
		}
	}

	return nil, errPriceNotFound
}

var yearRe = regexp.MustCompile(`^\d\d\d\d`)

func parseYear(data string) (uint, error) {
	rawYear := yearRe.FindString(data)
	if rawYear == "" {
		return 0, errors.Errorf("could not parse year from %s", data)
	}

	val, err := strconv.ParseUint(rawYear, 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse year as uint from %s", rawYear)
	}

	return uint(val), nil
}

var mileageRe = regexp.MustCompile(`^([\d,]+) miles?$`)

func parseMileage(data string) (uint, error) {
	matches := mileageRe.FindStringSubmatch(data)
	if len(matches) != 2 {
		return 0, errors.Errorf("could not parse %v as a mileage", data)
	}

	rawMileage := strings.ReplaceAll(matches[1], ",", "")
	val, err := strconv.ParseUint(rawMileage, 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse mileage as uint from %s", rawMileage)
	}

	return uint(val), nil
}

var engineRe = regexp.MustCompile(`^(\d)\.(\d)L$`)

func parseEngineSize(data string) (uint, error) {
	matches := engineRe.FindStringSubmatch(data)
	if len(matches) != 3 {
		return 0, errors.Errorf("could not parse %v as an engine size", data)
	}

	litres, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse %s as uint", matches[1])
	}

	var ccs uint64
	if matches[2] == "" {
		ccs = 0
	} else {
		ccs, err = strconv.ParseUint(matches[2], 10, 32)
		if err != nil {
			return 0, errors.Wrapf(err, "could not parse %s as uint", matches[2])
		}
	}

	val := (1000 * litres) + (100 * ccs)

	return uint(val), nil
}

func noErr(err error) {
	if err != nil {
		panic(err)
	}
}
