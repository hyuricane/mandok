window.addEventListener('alpine:init', () => {
  Alpine.data('projectContentEdit', (project, format, value) => ({
    project,
    format,
    value,
    init() {
      window.initMonacoEditor('project-detail', this.format, 'docker-compose', this.value)
      this.$watch('format', async format => {
        await window.initMonacoEditor('project-detail', format, 'docker-compose')
      });
    },
    submit() {
      console.log('project content edit submitted');
      const markers = window.monacoErrorMarkers()
      if (markers.length > 0) {
        return
      }
      const body = {
        format: this.format,
      }
      const editor = window.getMonacoEditor()
      if (editor) {
        body[this.format] = editor.getValue()
      }
      $ajax('/ax/project/' + this.name + '/edit', {
        method: 'POST',
        body,
        target: 'page',
        push: true,
      })
    }
  }));
  Alpine.data('serviceContentEdit', (project, service, format, value) => ({
    project,
    service,
    format,
    value,
    init() {
      window.initMonacoEditor('service-detail', this.format, 'service', this.value)
      this.$watch('format', async format => {
        await window.initMonacoEditor('service-detail', format, 'service')
      });
    },
    submit() {
      console.log('service content edit submitted');
      const markers = window.monacoErrorMarkers()
      if (markers.length > 0) {
        return
      }
      const body = {
        format: this.format,
      }
      const editor = window.getMonacoEditor()
      if (editor) {
        body[this.format] = editor.getValue()
      }
      $ajax('/ax/project/' + this.project + '/service/' + this.service + '/edit', {
        method: 'POST',
        body,
        target: 'page',
        push: true,
      })
    }
  }))
})