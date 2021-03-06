package waveguide

import "html/template"

var tmpl = template.Must(template.New("").Parse(`
{{define "header"}}
<!DOCTYPE html>
        <head>
                <title>Waveguide</title>
		<meta name="viewport" content="initial-scale=1.0">
		<meta charset="utf-8">
                <style>
			html, body {
				height: 100%;
			}
			body {
				font-family: monospace;
			}
                        table {
                                border-collapse: separate;
                                font-size: 12pt;
                        }
                        th {
                                text-align: left;
                        }
                        th, td {
                                padding: 0 1em 0.5ex 0;
                        }
                        form {
                        	margin: 0
                        }
			#map {
				height: 100%;
			}
                </style>
        </head>
        <body>
{{end}}

{{define "footer"}}
	</body>
</html>
{{end}}

{{define "root"}}
{{template "header"}}
                <table>
                	{{if .Spots}}
				<thead>
					<th>Location</th>
					<th>Coordinates</th>
					<th>Conditions</th>
					<th>Wave Height</th>
					<th>Last Updated</th>
				</thead>
				<tbody>
					{{range .Spots}}
						<tr>
							<td><a href="{{.MapURL}}">{{.HTMLName}}</a></td>
							<td>
								{{if .HasCoordinates}}
									<a href="{{.ClearCoordsURL}}">❌</a>
									<a href="{{.GoogleMapsURL}}">{{.FormattedCoordinates}}</a>
								{{else}}
									<form action="/coords" method="post">
										<input type="hidden" name="path" value="{{.MswPath}}" />
										<input name="coordinates" />
										<button type="submit">Submit</button>
									</form>
								{{end}}
							</td>
							<td><a href="{{.ReportURL}}">{{.Cond.Stars}}</a></td>
							<td>{{.Cond.WaveHeight}}</td>
							<td>{{.Cond.HowLong}} ago</td>
						</tr>
					{{end}}
				</tbody>
			{{else}}
				There's no data yet. You can get some by visiting <a href="/update_all">/update_all</a>.
			{{end}}
                </table>
{{template "footer"}}
{{end}}

{{define "action_response"}}
{{template "header"}}
		<div><a href="/">← home</a></div>
		<div id="message">{{.Message}}</div>
{{template "footer"}}
{{end}}

{{define "map"}}
{{template "header"}}
		<style>
			html, body {
				margin: 0;
				padding: 0;
			}
		</style>
		<div id="map"></div>
		<script>
var map;

var addSpot = function(s) {
	if (s.lat == 0 && s.lng == 0) {
		return;
	}
	var latLng = {lat: s.lat, lng: s.lng}
	var marker = new google.maps.Marker({
		position: latLng,
		map: map,
		title: s.title,
	});
	var infowindow = new google.maps.InfoWindow({
		content: s.content,
		map: map,
		position: latLng,
	});
	infowindow.close();
	marker.addListener('click', function() {
		infowindow.open(map, marker);
	});
};

function initMap() {
	var maui = {lat: 20.614489, lng: -156.439202};
	map = new google.maps.Map(document.getElementById('map'), {
		center: maui,
		zoom: 2,
		mapTypeId: google.maps.MapTypeId.HYBRID,
	});

	{{range .}}
		var s = {
			title: '{{.Name}}: {{.Cond.Stars}}',
			content: ('<div>{{.Name}}</div>' +
				  {{.StarsLinkToReport}} +
				  '<div>Waves {{.Cond.WaveHeight}}</div>' +
				  '<div>Updated {{.Cond.HowLong}} ago</div>'),
			lat: {{.Coordinates.Lat}},
			lng: {{.Coordinates.Lng}},
		};
		addSpot(s);
	{{end}}
}
		</script>
		<script src="https://maps.googleapis.com/maps/api/js?key=AIzaSyDZ8Bm6MbFrfZ37ko8UTCDErLVQa5DBn8M&callback=initMap" async defer></script>
{{template "footer"}}
{{end}}
`))
