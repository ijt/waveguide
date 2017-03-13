package waveguide

import "html/template"

var head = template.HTML(`
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
                </style>
        </head>
`)

var rootTmpl = template.Must(template.New("root").Parse(`
<html>
{{.Head}}
        <body>
                <table>
                	{{if .Spots}}
				<thead>
					<th>Location</th>
					<th>Conditions</th>
					<th>Wave Height</th>
					<th>Last Updated</th>
				</thead>
				<tbody>
					{{range .Spots}}
						<tr>
							<td><a href="{{.MapURL}}">{{.HTMLName}}</a></td>
							{{if .LatestQuality}}
								{{ $q := .LatestQuality }}
								<td><a href="{{.ReportURL}}">{{$q.Stars}}</a></td>
								<td>{{$q.WaveHeight}}</td>
								<td>{{$q.HowLong}} ago</td>
							{{end}}
						</tr>
					{{end}}
				</tbody>
			{{else}}
				There's no data yet. You can get some by visiting <a href="/update_all">/update_all</a>.
			{{end}}
                </table>
        </body>
</html>
`))
