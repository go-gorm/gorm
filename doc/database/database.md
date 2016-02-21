## Database

{% for section in book.chapters["database/database.md"].sections %}
* [**{{section.name}}**](../{{section.path}})
{% endfor %}
{{book.chapters}}

<script>
window.chapters = {}
{% for path, chapter in book.chapters %}
  chapters["{{path}}"] = {}
  chapters["{{path}}"]["name"] = "{{chapter.name}}"
  chapters["{{path}}"]["sections"] = [];
  {% for section in chapter.sections %}
  var section = {
    "name": "{{section.name}}",
    "path": "{{section.path}}",
  }
  chapters["{{path}}"]["sections"].push(section);
  {% endfor %}
{% endfor %}
</script>
