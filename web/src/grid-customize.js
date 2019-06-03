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

window.addEventListener('load', function() {
	const buffer = 100
	const notes = document.getElementById('notes')
	let remainingEl = null
	const checkRemaining = function() {
		const remainder = SqMGR.NotesMaxLength - this.value.length
		if (remainder <= buffer) {
			if (!remainingEl) {
				remainingEl = document.createElement('div')
				remainingEl.classList.add('remaining')
				this.parentNode.insertBefore(remainingEl, this.nextSibling)
			}

			remainingEl.textContent = remainder
		} else {
			if (remainingEl) {
				remainingEl.remove()
				remainingEl = null
			}
		}
	}

	notes.onkeyup = notes.onpaste = checkRemaining
	checkRemaining.apply(notes)

	const homeTeamName = document.getElementById('home-team-name')
	const awayTeamName = document.getElementById('away-team-name')
    const gridName = document.getElementById('grid-name')
    homeTeamName.oninput = awayTeamName.oninput = element => {
	    gridName.textContent = awayTeamName.value + ' vs. ' + homeTeamName.value

		if (!element) {
			return
		}

		const matchedTeams = TeamColors(element.target.value)
		if (matchedTeams.length === 0) {
			return
		}

       	console.log(TeamColors(element.target.value))
	}
	homeTeamName.oninput(null)
})
