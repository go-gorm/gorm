# Summary

* [GORM Guides](http://github.com/jinzhu/gorm)

* [Getting Started with GORM](README.md)
{% for path, chapter in book.chapters %}
* [{{ chapter.name }}]({{ path }})
  {% for section in chapter.sections %}
  * [{{ section.name }}]({{ section.path }})
  {% if section["sections"] %}{% for subsection in section.sections %}
    * [{{ subsection.name }}]({{ subsection.path }})
  {% endfor %}{% endif %}{% endfor %}
{% endfor %}
