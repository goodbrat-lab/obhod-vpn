"use strict";
"require view";
"require form";
"require baseclass";
"require network";
"require view.obhod.main as main";

// Settings content
"require view.obhod.settings as settings";

// Sections content
"require view.obhod.section as section";

// Dashboard content
"require view.obhod.dashboard as dashboard";

// Diagnostic content
"require view.obhod.diagnostic as diagnostic";

const EntryPoint = {
  async render() {
    main.injectGlobalStyles();

    const obhodMap = new form.Map(
      "obhod",
      _("Obhod Settings"),
      _("Configuration for Obhod service"),
    );
    // Enable tab views
    obhodMap.tabbed = true;

    // Sections tab (NOW INCLUDES SUBSCRIPTIONS)
    const sectionsSection = obhodMap.section(
      form.TypedSection,
      "section",
      _("Sections"),
    );
    sectionsSection.anonymous = false;
    sectionsSection.addremove = true;
    sectionsSection.template = "cbi/simpleform";

    // Render section content
    section.createSectionContent(sectionsSection);

    // Settings tab
    const settingsSection = obhodMap.section(
      form.TypedSection,
      "settings",
      _("Settings"),
    );
    settingsSection.anonymous = true;
    settingsSection.addremove = false;
    settingsSection.cfgsections = function () {
      return ["settings"];
    };

    // Render settings content
    settings.createSettingsContent(settingsSection);

    // Diagnostic tab
    const diagnosticSection = obhodMap.section(
      form.TypedSection,
      "diagnostic",
      _("Diagnostics"),
    );
    diagnosticSection.anonymous = true;
    diagnosticSection.addremove = false;
    diagnosticSection.cfgsections = function () {
      return ["diagnostic"];
    };

    // Render diagnostic content
    diagnostic.createDiagnosticContent(diagnosticSection);

    // Dashboard tab
    const dashboardSection = obhodMap.section(
      form.TypedSection,
      "dashboard",
      _("Dashboard"),
    );
    dashboardSection.anonymous = true;
    dashboardSection.addremove = false;
    dashboardSection.cfgsections = function () {
      return ["dashboard"];
    };

    // Render dashboard content
    dashboard.createDashboardContent(dashboardSection);

    // Inject core service
    main.coreService();

    return obhodMap.render();
  },
};

return view.extend(EntryPoint);
