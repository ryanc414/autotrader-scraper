package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const (
	baseUrl     = "https://www.autotrader.co.uk/car-search"
	maxNumPages = 100
)

func main() {
	queryOptions, outputFilename := parseArgs()
	cars := getAllCars(queryOptions)
	fmt.Printf("parsed info on %d cars\n", len(cars))

	err := writeOutput(outputFilename, cars)
	noErr(err)
}

func parseArgs() (*queryOptions, string) {
	opts := new(queryOptions)

	flag.StringVar(&opts.postcode, "postcode", "E144AD", "postcode for search")
	flag.StringVar(&opts.make, "make", "FORD", "make of car")
	flag.StringVar(&opts.model, "model", "FOCUS", "model of car")
	flag.Uint64Var(&opts.priceTo, "price-to", 25000, "price upper limit")
	flag.StringVar(&opts.bodyType, "body-type", "Hatchback", "body type")
	flag.StringVar(&opts.transmission, "transmission", "Manual", "transmission type")
	flag.Uint64Var(&opts.yearFrom, "year-from", 2015, "earliest year of manufacture")

	var outputFile string
	flag.StringVar(&outputFile, "output-filename", "car_info.json", "output filename")

	flag.Parse()

	return opts, outputFile
}

type queryOptions struct {
	postcode     string
	make         string
	model        string
	priceTo      uint64
	bodyType     string
	transmission string
	yearFrom     uint64
}

type CarInfo struct {
	Price      uint `json:"price"` // in pence
	Year       uint `json:"year"`
	Mileage    uint `json:"mileage"`
	EngineSize uint `json:"engine_size"` // CC
}

func getAllCars(opts *queryOptions) []*CarInfo {
	var allCars []*CarInfo

	for i := uint64(0); i < maxNumPages; i++ {
		cars, err := getPage(opts, i)
		if err != nil {
			fmt.Printf("error getting page %d: %s\n", i, err.Error())
		}
		allCars = append(allCars, cars...)
	}

	return allCars
}

func writeOutput(outputFilename string, cars []*CarInfo) error {
	data, err := json.Marshal(cars)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputFilename, data, 0644)
}

func getPageUrl(opts *queryOptions, pageNum uint64) (*url.URL, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("postcode", opts.postcode)
	q.Set("make", opts.make)
	q.Set("model", opts.model)
	q.Set("price-to", strconv.FormatUint(opts.priceTo, 10))
	q.Set("include-delivery-option", "on")
	q.Set("body-type", "Hatchback")
	q.Set("transmission", opts.transmission)
	q.Set("year-from", strconv.FormatUint(opts.yearFrom, 10))
	q.Set("onesearchad", "Used,Nearly New,New")
	q.Set("advertising-location", "at-cars")
	q.Set("page", strconv.FormatUint(pageNum, 10))
	u.RawQuery = q.Encode()

	return u, nil
}

func getPage(opts *queryOptions, pageNum uint64) ([]*CarInfo, error) {
	pageUrl, err := getPageUrl(opts, pageNum)
	if err != nil {
		return nil, errors.Wrap(err, "while getting URL for page")
	}

	rsp, err := http.Get(pageUrl.String())
	if err != nil {
		return nil, errors.Wrapf(err, "while making HTTP request to %s", pageUrl)
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
