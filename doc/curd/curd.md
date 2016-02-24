# CRUD: Reading and Writing Data

{% for section in book.chapters["curd/curd.md"].sections %}
* [**{{section.name}}**](../{{section.path}})
{% if section["sections"] %}{% for subsection in section.sections %}
  * [**{{ subsection.name }}**]({{ subsection.path }})
{% endfor %}{% endif %}
{% endfor %}

