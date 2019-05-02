/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

SqMGR.Config = {
	Types: {
		'std100': 100,
		'std25': 25
	}
}

SqMGR.buildSquares = function() {
	const grid = SqMGR.grid

	new SqMGR.GridBuilder(grid)
}

SqMGR.GridBuilder = function(grid) {
	this.modal = new SqMGR.Modal()
	this.grid = grid
	this.templates = document.querySelector('section.templates')
	this.templates.remove()

	this.draw(null)
	this.loadSquares()
}


SqMGR.GridBuilder.prototype.draw = function(squares) {
	let container = document.getElementById('grid-container'),
		parent = document.createElement('div'),
		i, elem, elem2, numSquares,
		square

	parent.classList.add('squares')
	parent.classList.add(this.grid.gridType)

	elem = document.createElement('div')
	elem.classList.add('spacer')
	parent.appendChild(elem)

	;["Home", "Away"].forEach(function (team) {
		elem = document.createElement('div')
		elem.classList.add('team')
		elem.classList.add(team.toLowerCase()+ '-team')
		elem.style.setProperty('--team-primary', this.getTeamValue(team, "Color1"))
		elem.style.setProperty('--team-secondary', this.getTeamValue(team, "Color2"))
		elem.style.setProperty('--team-tertiary', this.getTeamValue(team, "Color3"))
		elem2 = document.createElement('span')
		elem2.textContent = this.getTeamValue(team, "Name")
		elem.appendChild(elem2)
		parent.appendChild(elem)

		for (i=0; i<10; i++) {
			elem = document.createElement('div')
			elem.classList.add('score')
			elem.classList.add(team.toLowerCase() + '-score')
			elem.classList.add(team.toLowerCase() + '-score-'+i)
			elem2 = document.createElement('span')
			// FIXME: will need to figure out how to handle scores
			elem2.textContent = ''
			elem.appendChild(elem2)
			parent.appendChild(elem)
		}
	}.bind(this))

	numSquares = SqMGR.Config.Types[this.grid.gridType]
	for (i=0; i<numSquares; i++) {
		square = squares ? squares[i] : null

		elem = document.createElement('div')
		elem.onclick = this.clickSquare.bind(this, i)
		elem.classList.add('square')
		if (square) {
            elem.classList.add(square.state)
        }
		elem.setAttribute('data-sqid', i)

		// add the square id
		elem2 = document.createElement('span')
		elem2.textContent = i
		elem2.classList.add('square-id')
		elem.appendChild(elem2)

		// add the name
		elem2 = document.createElement('span')
		elem2.classList.add('name')

		if (square) {
			elem2.textContent = square.claimant
		}

		elem.appendChild(elem2)
		parent.appendChild(elem)
	}

	container.innerHTML = ''
	container.appendChild(parent)
}

SqMGR.GridBuilder.prototype.loadSquares = function() {
	const container = document.getElementById('grid-container')
	container.classList.add('loading')

	SqMGR.get("/grid/" + this.grid.token + "/squares", function (data) {
		this.draw(data)
		container.classList.remove('loading')
	}.bind(this))
}

SqMGR.GridBuilder.prototype.getTeamValue = function(team, prop) {
	const setting = team.toLowerCase() + "Team" + prop
	return this.grid.settings[setting]
}

SqMGR.GridBuilder.prototype.clickSquare = function(squareID) {
    const path = "/grid/" + this.grid.token + "/squares/" + squareID
	SqMGR.get(path, function(data) {
		const squareDetails = this.templates.querySelector('div.square-details').cloneNode(true)
        squareDetails.querySelector('td.square-id').textContent = data.squareID
		squareDetails.querySelector('td.claimant').textContent = data.claimant
		squareDetails.querySelector('td.state').textContent = data.state
		squareDetails.querySelector('td.modified').setAttribute('data-datetime', data.modified)

		const auditLog = squareDetails.querySelector('section.audit-log')

        if (data.logs) {
			const auditLogTbody = auditLog.querySelector('tbody')
			const auditLogRowTpl = auditLog.querySelector('tr.template')
			auditLogRowTpl.remove()

			data.logs.forEach(function (log) {
				const row = auditLogRowTpl.cloneNode(true)
				row.querySelector('td.created').setAttribute('data-datetime', log.created)
				row.querySelector('td.state').textContent = log.state
				row.querySelector('td.remote-addr').textContent = log.remoteAddr
				row.querySelector('td.note').textContent = log.note

				auditLogTbody.appendChild(row)
			}.bind(this))
		} else {
        	auditLog.remove()
		}

		SqMGR.DateTime.format(squareDetails)

		this.modal.show(squareDetails)
	}.bind(this))
}

SqMGR.get = function(path, callback, errorCallback) {
	const xhr = new XMLHttpRequest()
	xhr.open("GET", path)
	xhr.onload = function() {
	    let data
		try {
	   		data = JSON.parse(xhr.response)
		} catch (err) {
	    	console.log("could not parse JSON", err)
			return
		}

		if (data.status === "OK") {
			callback(data.result)
		} else if (typeof(errorCallback) === "function") {
			errorCallback(data)
		}
	}
	xhr.send()
}

window.addEventListener('load', SqMGR.buildSquares)
