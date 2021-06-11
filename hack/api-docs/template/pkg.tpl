{{ define "packages" }}
---
layout: default
title: API Reference 
nav_order: 3
---
    <h1>Orkestra API Reference</h1>

    {{ with .packages}}
        <p>Packages:</p>
        <ul class="simple">
            {{ range . }}
                <li>
                    <a href="#{{- packageAnchorID . -}}">{{ packageDisplayName . }}</a>
                </li>
            {{ end }}
        </ul>
    {{ end}}

    {{ range .packages }}
        <h2 id="{{- packageAnchorID . -}}">
            {{- packageDisplayName . -}}
        </h2>

        {{ with (index .GoPackages 0 )}}
            {{ with .DocComments }}
                {{ safe (renderComments .) }}
            {{ end }}
        {{ end }}

        <h3>Resource Types:</h3>

        <ul class="simple">
            {{- range (visibleTypes (sortedTypes .Types)) -}}
                {{ if isExportedType . -}}
                    <li>
                        <a href="{{ linkForType . }}">{{ typeDisplayName . }}</a>
                    </li>
                {{- end }}
            {{- end -}}
        </ul>

        {{ range (visibleTypes (sortedTypes .Types))}}
            {{ template "type" .  }}
        {{ end }}
    {{ end }}

    <p class="last">This page was automatically generated with <code>gen-crd-api-reference-docs</code></p>
{{ end }}
