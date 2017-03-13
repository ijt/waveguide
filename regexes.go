package waveguide

import "regexp"

var starSectionRx = regexp.MustCompile(`<ul class="rating rating-large clearfix">.*?</ul>`)
var starRx = regexp.MustCompile(`<li class="active"> *<i class="glyphicon glyphicon-star"></i> *</li>`)
var heightRx = regexp.MustCompile(`(\d+(?:-\d+)?)<small>ft`)
var reportRx = regexp.MustCompile(`/[^"/]+-Surf-Report/\d+/`)
var srpTailRx = regexp.MustCompile(`-Surf-Report/\d+/`)
var coordsRx = regexp.MustCompile(`(-?\d+\.?\d*),\s*(-?\d+\.?\d*)`)
