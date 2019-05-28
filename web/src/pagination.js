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

export default class Pagination {
	constructor(nav) {
		let parentNode

		for (parentNode = nav.parentNode; parentNode.nodeName !== '#document' && !parentNode.getAttribute("data-pagination"); parentNode = parentNode.parentNode)
			;

		if (parentNode.nodeName === '#document') {
			return
		}

		nav.querySelectorAll('a').forEach(function(link) {
			link.onclick = function() {
				const request = new XMLHttpRequest()
				request.open("GET", link.getAttribute("href"))
				request.onload = function() {
					const div = document.createElement('div')
					div.innerHTML = request.responseText
					new Pagination(div.querySelector('nav.pagination'))
					parentNode.replaceWith(div.firstElementChild)
				}
				request.send()

				return false
			}
		})
	}
}
