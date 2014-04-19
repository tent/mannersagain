# mannersagain

Package mannersagain combines [manners](https://github.com/braintree/manners)
and [goagain](https://github.com/rcrowley/goagain) to provide graceful hot
restarting of `net/http` servers.

To use it, just replace your call to `http.ListenAndServe` with a call to
`mannersagain.ListenAndServe`. When you send SIGUSR2 to the process, it will
pass the listener to a new process and exit gracefully. When you send SIGQUIT to
the process it will exit gracefully.

mannersagain supports both of the `Single` and `Double` strategies that goagain
provides, set `goagain.Strategy` before calling `ListenAndServe` to change the
default.
