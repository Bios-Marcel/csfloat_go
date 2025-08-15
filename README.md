## Unofficial CSFloat API wrapper

This is a work in progress API wrapper for [CSFloat](https://csfloat.com).

It is not 100% feature complete. If you need any additional endpoints, either
make a PR or create an issue.

The API *might* not be 100% stable and could change at any time.

There's no extensive test suite, as it is hard to test this without potentially
causing chaos in my own account.

Given that there is *NO* documentation for CSFloat, everything here was reversed
through the browser.

## Concepts

### Ratelimits

CSFloat has two types of ratelimiting:

1. IP-based (separate for IPv4 and IPv6)
  > N request per 5 minutes
2. Account-based per endpoint (includes API Key)
  > Each endpoint will tell you

This wrapepr *does not* respect ratelimits on its own, but exposes a
`Ratelimits` field on every endpoint, which you can use to respect them
yourself.

### Errors

CSFloat uses a generic error format. These errors are exposed in the `Error`
field of each response and need to be checked for `nil` before accessing.

I do *NOT* know all error codes yet, so there are only constants for the ones
I stumbled upon.

## Known issues

### Timeouts

Sometimes CSFloat will have very slow reponse times, where you don't even
receive the headers. This can cause timeouts sometimes, even though the actual
action behind the request is already done.

For example you click buy on an item and it gives you a timeout error, but the
purchase has alreadt completed / will keep completing in the background.
