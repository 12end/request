module github.com/12end/request

go 1.20

require (
	github.com/12end/tls v0.0.0-20230329031950-bbfc948c6240
	github.com/valyala/fasthttp v1.46.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
)

replace github.com/valyala/fasthttp v1.46.0 => github.com/12end/fasthttp v0.0.0-20230427071457-646400e30865
