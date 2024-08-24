FIND method_declaration AS md
WHERE md.getName() == "onCreate" && md.getVisibility() == "protected"
