importScripts('wasm_exec.js')

function loadAgent(wasm) {
    
    const go = new Go()
    WebAssembly.instantiateStreaming(fetch(wasm), go.importObject).then(({ instance }) => go.run(instance))

}

addEventListener('install', (event) => {
    event.waitUntil(skipWaiting())
})

addEventListener('activate', event => {
    event.waitUntil(clients.claim())
})

loadAgent('agent.wasm')
