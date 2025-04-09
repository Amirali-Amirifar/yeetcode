import * as monaco from "monaco-editor"
function createEditor(containerId, initialCode = "", language = "go") {
    return monaco.editor.create(document.getElementById(containerId), {
        value: initialCode,
        language,
        theme: "vs-dark", // You can customize this later
        fontSize: 16,
        minimap: {enabled: false},
        wordWrap: "on",
    });
}

window.highlight = {
    createEditor,
};