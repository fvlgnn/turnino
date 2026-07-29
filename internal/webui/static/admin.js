(() => {
  "use strict";

  const presets = {
    "numeric-2": {numbering_mode: "fixed", prefix: "", alphabetic_start: "", alphabetic_end: "", digits: 2, min_value: 0, max_value: 99},
    "numeric-3": {numbering_mode: "fixed", prefix: "", alphabetic_start: "", alphabetic_end: "", digits: 3, min_value: 0, max_value: 999},
    "a-e": {numbering_mode: "progressive", prefix: "", alphabetic_start: "A", alphabetic_end: "E", digits: 2, min_value: 0, max_value: 99},
    "a-j": {numbering_mode: "progressive", prefix: "", alphabetic_start: "A", alphabetic_end: "J", digits: 2, min_value: 0, max_value: 99},
  };

  const updatePolicyFields = (form, preset = "custom") => {
    const mode = form.elements.numbering_mode?.value;
    const presetIsAlphabetic = preset === "a-e" || preset === "a-j";
    form.querySelectorAll("[data-fixed-prefix-field]").forEach((field) => {
      field.hidden = presetIsAlphabetic || (preset === "custom" && mode === "progressive");
    });
    form.querySelectorAll("[data-alphabetic-field]").forEach((field) => {
      field.hidden = mode !== "progressive";
    });
  };

  document.querySelectorAll("[data-numbering-preset]").forEach((select) => {
    const form = select.closest("form");
    if (!form) return;
    select.addEventListener("change", () => {
      const values = presets[select.value];
      if (values) {
        Object.entries(values).forEach(([name, value]) => {
          const field = form.elements[name];
          if (field) field.value = value;
        });
      }
      updatePolicyFields(form, select.value);
    });
    form.elements.numbering_mode?.addEventListener("change", () => {
      select.value = "custom";
      updatePolicyFields(form, select.value);
    });
    updatePolicyFields(form, select.value);
  });

  document.querySelectorAll(".queue-policy-form").forEach((form) => {
    const mode = form.elements.numbering_mode;
    mode?.addEventListener("change", () => updatePolicyFields(form));
    updatePolicyFields(form);
  });
})();
