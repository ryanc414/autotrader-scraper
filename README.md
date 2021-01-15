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

By default the scraper will fetch data on Ford Focus models made in
2015 or newer, with various other constraints (e.g. manual transmission only).
Make, model and other search details can be tweaked using the CLI flags, run
`go run scraper.go -h` more more details. Similarly, the plot.py script assumes
by default it is plotting data for 2015+ Ford Focuses but this can be overridden
via CLI args (`pipenv run python plot.py -h` for the details there).