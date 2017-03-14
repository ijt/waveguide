package waveguide

import "html/template"

var tmpl = template.Must(template.New("").Parse(`
{{define "header"}}
<html>
        <head>
                <title>Waveguide</title>
                <style>
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
									<a href="{{.MapsURL}}">{{.FormattedCoordinates}}</a>
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
<!DOCTYPE html>
<html>
  <head>
    <title>Simple Map</title>
    <meta name="viewport" content="initial-scale=1.0">
    <meta charset="utf-8">
    <style>
      /* Always set the map height explicitly to define the size of the div
       * element that contains the map. */
      #map {
        height: 100%;
      }
      /* Optional: Makes the sample page fill the window. */
      html, body {
        height: 100%;
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <div id="map"></div>
    <script>
      var map;
      function initMap() {
        map = new google.maps.Map(document.getElementById('map'), {
          center: {lat: -34.397, lng: 150.644},
          zoom: 8
        });
      }
    </script>
    <script src="https://maps.googleapis.com/maps/api/js?key=AIzaSyDZ8Bm6MbFrfZ37ko8UTCDErLVQa5DBn8M&callback=initMap"
    async defer></script>
  </body>
</html>
{{end}}
`))
