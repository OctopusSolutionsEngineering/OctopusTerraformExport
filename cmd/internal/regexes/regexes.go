package regexes

import "regexp"

var FeedRegex = regexp.MustCompile("Feeds-\\d+")
var AccountRegex = regexp.MustCompile("Accounts-\\d+")
var GitCredentialsRegex = regexp.MustCompile("GitCredentials-\\d+")
var CertificatesRegex = regexp.MustCompile("Certificates-\\d+")
var WorkerPoolsRegex = regexp.MustCompile("WorkerPools-\\d+")
var ProjectsRegex = regexp.MustCompile("Projects-\\d+")
