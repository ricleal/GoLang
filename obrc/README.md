# One Billion Row Challenge in Golang


https://github.com/gunnarmorling/1brc


## In:

`<station name>;<temperature>`

```
Yellowknife;16.0
Entebbe;32.9
Porto;24.4
Vilnius;12.4
Fresno;7.9
Maun;17.5
Panama City;39.5
...
```

format `<station name>;<temperature>`

temperature is a floating-point number ranging from -99.9 to 99.9 with precision limited to one decimal point.


## Out:

```
{Abha=-23.0/18.0/59.2, Abidjan=-16.2/26.0/67.3, Abéché=-10.0/29.4/69.0, ...}
```

format `{<station name>=<min>/<mean/<max>, ...}`

The expected output format is sorted alphabetically by station name, and where min, mean and max denote the computed minimum, average and maximum temperature readings for each respective station.

## Create sample data

```bash
gcc create-sample.c -lm  -o create-sample
```

## Profiling

```bash
go run ./obrc -cpuprofile cpu.prof -memprofile mem.prof > /dev/null                                                                                                                                                                         ──(Tue,Mar26)─┘

go tool pprof -http=":8080" ./cpu.prof 
```

TODO: INCOMPLETE