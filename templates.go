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
`))
