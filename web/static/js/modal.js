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

SqMGR.Modal = function(optionalParent) {
    this.parent = optionalParent // the parent modal (optional)
    this.node = null
    this.nestedModal = null
    this._keyup = this.keyup.bind(this)
}

SqMGR.Modal.prototype.nest = function() {
    if (this.nestedModal) {
        this.nestedModal.close()
    }

    this.nestedModal = new SqMGR.Modal(this)
    return this.nestedModal
}

SqMGR.Modal.prototype.nestedDidClose = function() {
    this.nestedModal = null
}

SqMGR.Modal.prototype.close = function() {
    window.removeEventListener('keyup', this._keyup)

    if (this.node) {
        this.node.dispatchEvent(new Event('modalclose'))

        this.node.remove()
        this.node = null
    }

    if (this.parent) {
        this.parent.nestedDidClose()
    }
}

SqMGR.Modal.prototype.show = function(childNode) {
    const node = document.createElement('div')
    node.classList.add('modal')

    const closeLink = document.createElement('a')
    closeLink.setAttribute('href', '#')
    closeLink.classList.add('close')

    const closeText = document.createElement('span')
    closeText.textContent = 'Close'

    const container = document.createElement('div')
    container.classList.add('container')

    closeLink.appendChild(closeText)
    container.appendChild(closeLink)
    container.appendChild(childNode)
    node.appendChild(container)

    container.onclick = function(event) {
        event.cancelBubble = true
    }

    if (this.node) {
        this.close()
    }

    this.node = node

    this.node.onclick = closeLink.onclick = this.close.bind(this)

    document.body.appendChild(node)

    window.addEventListener('keyup', this._keyup)

    return node
}

SqMGR.Modal.prototype.showError = function(errorMsg) {
    const div = document.createElement('div')
    div.classList.add('error')
    div.textContent = errorMsg

    this.show(div)
}


SqMGR.Modal.prototype.keyup = function(event) {
    if (this.nestedModal) {
        return
    }

    if (event.key === 'Escape') {
        event.stopPropagation()
        this.close()
        return
    }
}