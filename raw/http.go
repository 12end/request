package raw

// DefaultClient is the default HTTP client for doing raw requests
var DefaultClient = Client{
	dialer:  new(dialer),
	Options: DefaultOptions,
}
