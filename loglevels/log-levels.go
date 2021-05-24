package loglevels

// - `0/Info`: stuff that the user should be able to see in the logs, always
// - `2/Error`: stuff that broke, that we need to tell the user in the logs, always
// - `5/Verbose`: stuff that is sometimes handy to know.
// - `7/Flow`: program flow
// - `9/Trace`: program flow with full details, for debugging nasty errors

const Info int = 0
const Error int = 2
const Verbose int = 5
const Flow int = 7
const Trace int = 9
