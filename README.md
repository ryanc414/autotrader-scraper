Autotrader Scraper Tool
-----------------------

This is a super simple tool for scraping the Autotrader website for data on used
cars, analysing that data and plotting some graphs.

Prerequisites: requires go and python standard tooling (including pipenv for
python). Probably would have been simpler if I'd used python for everything, but
I'm more familiar with HTML parsing using the go libraries...

Run the scraper with `go run scraper.go`, if this is successful it will write
the data to `car_info.json` in the current working dir. This data can then be
analysed and displayed by running `pipenv run python plot.py`

Currently the scaper is hard coded to fetch data on Ford Focus models made in
2015 or newer, with various other constraints (e.g. manual transmission only).
This can be changed by editing the baseUrl constant in the scraper.go file. In
future I might make this more configurable via CLI args or something.