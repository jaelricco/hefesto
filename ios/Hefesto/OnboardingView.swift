import HefestoPlan
import HefestoSync
import SwiftUI

// The onboarding (ADR 0023): the questions of onboarding.md §3, page by
// page, then the first plan. It reads and fills HefestoPlan's form; which
// questions apply and what goes out is decided there, not here.

/// The onboarding from the first question to the first plan.
struct OnboardingView: View {
    @Environment(AppModel.self) private var app
    @Environment(\.dismiss) private var dismiss
    @State private var index = 0
    @State private var loadFailed = false
    @State private var sending = false
    @State private var failure: String?
    /// The result once nothing is left to ask: the plan, or why there is none.
    @State private var result: OnboardingOutcome?

    var body: some View {
        NavigationStack {
            Group {
                if let result {
                    OnboardingResultView(outcome: result) { dismiss() }
                } else if let catalogue = app.onboarding.catalogue {
                    pager(catalogue)
                } else {
                    loading
                }
            }
            .themedScreen()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if result == nil {
                    ToolbarItem(placement: .cancellationAction) { Button("Close") { dismiss() } }
                }
            }
        }
        .preferredColorScheme(.dark)
        .task {
            await load()
            // A demo opens on the page it shows.
            if let page = app.demo?.onboardingPage, let i = app.onboarding.pages.firstIndex(of: page) { index = i }
        }
    }

    private var loading: some View {
        VStack(spacing: 16) {
            if loadFailed {
                Text("The questions could not be loaded. Starting your plan needs a connection.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
                    .multilineTextAlignment(.center)
                Button("Try again") { Task { await load() } }
                    .buttonStyle(SecondaryButtonStyle())
            } else {
                ProgressView().tint(Palette.textSecondary)
            }
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func pager(_ c: OnboardingCatalogue) -> some View {
        let pages = app.onboarding.pages
        let i = min(index, pages.count - 1)
        let page = pages[i]
        return VStack(spacing: 0) {
            ProgressView(value: Double(i + 1), total: Double(pages.count))
                .tint(Palette.ember)
                .padding(.horizontal, 20)
                .padding(.vertical, 8)
                .accessibilityLabel(Text("Step \(i + 1) of \(pages.count)"))
            ScrollView {
                OnboardingPageView(page: page, catalogue: c, model: app.onboarding)
                    .padding(.horizontal, 20)
                    .padding(.vertical, 12)
            }
            .id(page)
            .scrollDismissesKeyboard(.interactively)
            if let failure {
                Text(verbatim: failure)
                    .font(.detailText)
                    .foregroundStyle(Palette.text)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 20)
                    .padding(.top, 8)
            }
            bottomBar(page, at: i, of: pages.count)
        }
    }

    private func bottomBar(_ page: OnboardingPage, at i: Int, of count: Int) -> some View {
        HStack(spacing: 10) {
            if i > 0 {
                Button {
                    failure = nil
                    index = i - 1
                } label: {
                    Image(systemName: "chevron.left")
                        .font(.system(size: 22, weight: .medium))
                        .foregroundStyle(Palette.text)
                        .frame(width: 64, height: 64)
                        .background(Palette.surfaceRaised, in: .rect(cornerRadius: 16))
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Back")
            }
            if page == .finish {
                Button {
                    Task { await submit() }
                } label: {
                    if sending {
                        ProgressView().tint(Palette.emberInk)
                    } else if app.onboarding.outcome?.status == .needsAnswers {
                        Label("Send answers", systemImage: "arrow.up")
                    } else {
                        Label("Create my plan", systemImage: "sparkles")
                    }
                }
                .buttonStyle(PrimaryButtonStyle())
                .disabled(sending || !clarified)
            } else {
                Button {
                    failure = nil
                    index = min(i + 1, count - 1)
                } label: {
                    Label("Next", systemImage: "chevron.right")
                }
                .buttonStyle(PrimaryButtonStyle())
                .disabled(!app.onboarding.isComplete(page))
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 8)
        .background(Palette.background)
    }

    /// Every open clarification has its answer.
    private var clarified: Bool {
        guard let outcome = app.onboarding.outcome, outcome.status == .needsAnswers else { return true }
        return outcome.questions.allSatisfy { app.onboarding.form.clarifications[$0.id] != nil }
    }

    private func load() async {
        loadFailed = false
        do {
            try await app.onboarding.load()
        } catch {
            loadFailed = true
        }
    }

    private func submit() async {
        sending = true
        failure = nil
        do {
            let outcome = try await app.onboarding.submit()
            if outcome.status != .needsAnswers {
                result = outcome
                await app.plan.refresh()
            }
        } catch {
            failure = OnboardingText.failure(error)
            if (error as? OnboardingError) == .alreadyOnboarded { await app.plan.refresh() }
        }
        sending = false
    }
}

/// One page's questions.
struct OnboardingPageView: View {
    let page: OnboardingPage
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        switch page {
        case .frame: OnboardingFramePage(catalogue: catalogue, model: model)
        case .goals: OnboardingGoalsPage(catalogue: catalogue, model: model)
        case .availability: OnboardingAvailabilityPage(model: model)
        case .equipment: OnboardingEquipmentPage(model: model)
        case .background: OnboardingBackgroundPage(catalogue: catalogue, model: model)
        case .performance: OnboardingPerformancePage(catalogue: catalogue, model: model)
        case .health: OnboardingHealthPage(catalogue: catalogue, model: model)
        case let .region(id): OnboardingRegionPage(regionId: id, catalogue: catalogue, model: model)
        case .finish: OnboardingFinishPage(catalogue: catalogue, model: model)
        }
    }
}

// MARK: - Pages

/// Block A: birth year, consent, the disclaimer.
struct OnboardingFramePage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            PageTitle(
                "Your plan",
                detail: "Hefesto plans your training from your answers. It takes about five minutes; after that, the planner learns from your log.")

            VStack(alignment: .leading, spacing: 10) {
                QuestionText("In which year were you born?")
                TextField("Year", value: $model.form.birthYear, format: .number.grouping(.never))
                    .keyboardType(.numberPad)
                    .font(.metricSmall)
                    .foregroundStyle(Palette.text)
                    .padding(.horizontal, 14)
                    .frame(minHeight: 52)
                    .background(Palette.surface, in: .rect(cornerRadius: 12))
            }

            VStack(alignment: .leading, spacing: 10) {
                QuestionText("May Hefesto keep your health answers?")
                Text("Complaints and health answers let the planner work around them. Without your consent the safety questions are still asked, but only used for this plan and not kept, and the planner plans conservatively.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
                ChoiceButton(
                    label: String(localized: "Yes, keep them"), selected: model.form.healthConsent == true
                ) { model.form.healthConsent = true }
                ChoiceButton(
                    label: String(localized: "No, don't keep them"), selected: model.form.healthConsent == false
                ) { model.form.healthConsent = false }
            }

            VStack(alignment: .leading, spacing: 12) {
                Text(verbatim: catalogue.disclaimer).font(.detailText).foregroundStyle(Palette.text)
                Toggle(isOn: $model.form.disclaimerAck) {
                    Text("I have read this.").font(.bodyMedium).foregroundStyle(Palette.text)
                }
                .tint(Palette.ember)
            }
            .card()
        }
    }
}

/// Block B: up to three goals, in order, each with its level and date.
struct OnboardingGoalsPage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 24) {
            PageTitle("Your goals", detail: "Choose up to three skills. The first one gets the most time.")
            VStack(spacing: 8) {
                ForEach(catalogue.goalSkills) { skill in row(skill) }
            }
            if !model.form.goals.isEmpty {
                CapsLabel("In this order")
                ForEach(Array(model.form.goals.enumerated()), id: \.element.id) { i, goal in
                    card(i, goal)
                }
            }
        }
    }

    private func row(_ skill: OnboardingCatalogue.Skill) -> some View {
        let position = model.form.goals.firstIndex { $0.skill == skill.slug }
        return Button {
            toggle(skill)
        } label: {
            HStack {
                Text(verbatim: skill.name)
                    .font(.rowTitle)
                    .foregroundStyle(position == nil ? Palette.text : Palette.emberInk)
                Spacer()
                if let position {
                    Text(verbatim: "\(position + 1)").font(.metricSmall).foregroundStyle(Palette.emberInk)
                }
            }
            .padding(.horizontal, 16)
            .frame(minHeight: 56)
            .background(position == nil ? Palette.surface : Palette.ember, in: .rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
        .disabled(position == nil && model.form.goals.count >= 3)
        .accessibilityAddTraits(position == nil ? [] : .isSelected)
    }

    private func toggle(_ skill: OnboardingCatalogue.Skill) {
        if let i = model.form.goals.firstIndex(where: { $0.skill == skill.slug }) {
            model.form.goals.remove(at: i)
        } else if model.form.goals.count < 3, let level = skill.firstMilestone {
            model.form.goals.append(.init(skill: skill.slug, targetLevel: level.slug))
        }
    }

    private func card(_ i: Int, _ goal: OnboardingForm.Goal) -> some View {
        let skill = catalogue.skill(goal.skill)
        return VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(verbatim: "\(i + 1)  \(skill?.name ?? goal.skill)").font(.rowTitle).foregroundStyle(Palette.text)
                Spacer()
                if i > 0 {
                    Button { model.form.goals.swapAt(i, i - 1) } label: { Image(systemName: "arrow.up") }
                        .buttonStyle(RoundIconButtonStyle(size: 44))
                        .accessibilityLabel("Move up")
                }
            }
            HStack {
                Text("Up to").font(.bodyMedium).foregroundStyle(Palette.textSecondary)
                Spacer()
                Picker("Up to", selection: level(i)) {
                    ForEach(skill?.levels ?? []) { Text(verbatim: $0.name).tag($0.slug) }
                }
                .labelsHidden()
                .themedPicker()
            }
            Toggle(isOn: hasDate(i)) {
                Text("By a date").font(.bodyMedium).foregroundStyle(Palette.text)
            }
            .tint(Palette.ember)
            if goal.targetDate != nil {
                DatePicker("Date", selection: date(i), in: Date.now..., displayedComponents: .date)
                    .font(.bodyMedium)
                    .tint(Palette.ember)
                Text("Only to show how long the way usually takes. The plan follows where you stand, not the date.")
                    .font(.fine)
                    .foregroundStyle(Palette.textSecondary)
            }
        }
        .card()
    }

    private func level(_ i: Int) -> Binding<String> {
        Binding(
            get: { model.form.goals.indices.contains(i) ? model.form.goals[i].targetLevel : "" },
            set: { if model.form.goals.indices.contains(i) { model.form.goals[i].targetLevel = $0 } })
    }

    private func hasDate(_ i: Int) -> Binding<Bool> {
        Binding(
            get: { model.form.goals.indices.contains(i) && model.form.goals[i].targetDate != nil },
            set: { on in
                guard model.form.goals.indices.contains(i) else { return }
                model.form.goals[i].targetDate = on ? DayFormat.string(Date.now.addingTimeInterval(180 * 86_400)) : nil
            })
    }

    private func date(_ i: Int) -> Binding<Date> {
        Binding(
            get: {
                guard model.form.goals.indices.contains(i), let day = model.form.goals[i].targetDate else { return .now }
                return DayFormat.date(day)
            },
            set: { if model.form.goals.indices.contains(i) { model.form.goals[i].targetDate = DayFormat.string($0) } })
    }
}

/// Block C: sessions a week and minutes a session.
struct OnboardingAvailabilityPage: View {
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            PageTitle("Your time", detail: "The plan fits into what you have. Rest days count towards the streak.")
            ChoiceQuestion(
                question: String(localized: "How many times a week can you train?"),
                choices: (1...7).map { Choice(id: "\($0)", label: "\($0)") },
                selection: number(\.sessionsPerWeek), columns: 4)
            ChoiceQuestion(
                question: String(localized: "How much time do you have per session?"),
                choices: OnboardingForm.sessionMinutesChoices.map { m in
                    Choice(id: "\(m)", label: m == 90 ? String(localized: "\(m)+ min") : String(localized: "\(m) min"))
                },
                selection: number(\.sessionMinutes), columns: 3)
        }
    }

    private func number(_ path: WritableKeyPath<OnboardingForm, Int?>) -> Binding<String?> {
        Binding(
            get: { model.form[keyPath: path].map { "\($0)" } },
            set: { model.form[keyPath: path] = $0.flatMap { Int($0) } })
    }
}

/// Block D: what the athlete has to train with.
struct OnboardingEquipmentPage: View {
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 24) {
            PageTitle("Your equipment", detail: "Choose everything you can use. A floor and a wall are always there.")
            LazyVGrid(columns: [GridItem(.flexible(), spacing: 8), GridItem(.flexible(), spacing: 8)], spacing: 8) {
                ForEach(OnboardingText.equipmentChoices, id: \.self) { key in
                    ChoiceButton(
                        label: OnboardingText.equipment(key), detail: OnboardingText.equipmentDetail(key),
                        selected: model.form.equipment.contains(key)
                    ) {
                        if model.form.equipment.contains(key) {
                            model.form.equipment.remove(key)
                        } else {
                            model.form.equipment.insert(key)
                        }
                    }
                }
            }
        }
    }
}

/// Block E: body weight, training level and age, the last regular training.
struct OnboardingBackgroundPage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            PageTitle("Your background", detail: "This sets where the plan starts and how fast it climbs.")
            VStack(alignment: .leading, spacing: 10) {
                QuestionText("How much do you weigh?")
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    TextField("Body weight", value: $model.form.bodyweightKg, format: .number.precision(.fractionLength(0...1)))
                        .keyboardType(.decimalPad)
                        .font(.metricSmall)
                        .foregroundStyle(Palette.text)
                    Text("kg").font(.meta).foregroundStyle(Palette.textSecondary)
                }
                .padding(.horizontal, 14)
                .frame(minHeight: 52)
                .background(Palette.surface, in: .rect(cornerRadius: 12))
            }
            ChoiceQuestion(
                question: String(localized: "How would you describe your training?"),
                choices: Choice.list(OnboardingText.trainingLevels), selection: $model.form.trainingLevel)
            ChoiceQuestion(
                question: String(localized: "How long have you trained with your body weight regularly?"),
                choices: Choice.list(OnboardingText.trainingAges), selection: $model.form.trainingAge)
            ChoiceQuestion(
                question: String(localized: "When did you last train regularly, at least once a week?"),
                choices: Choice.list(OnboardingText.lastRegular), selection: $model.form.lastRegular)
            if model.form.asksPreBreak {
                ForEach(model.form.goals) { goal in
                    if let skill = catalogue.skill(goal.skill) {
                        ChoiceQuestion(
                            question: String(localized: "What could you do before the break: \(skill.name)?"),
                            choices: skill.levels.map { Choice(id: $0.slug, label: $0.name) },
                            selection: preBreak(skill.slug))
                    }
                }
            }
        }
    }

    private func preBreak(_ skill: String) -> Binding<String?> {
        Binding(get: { model.form.preBreak[skill] }, set: { model.form.preBreak[skill] = $0 })
    }
}

/// Block F: where the athlete stands, counted cleanly.
struct OnboardingPerformancePage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            PageTitle(
                "Where you stand",
                detail: "Count only clean repetitions and seconds: the full range, no swing, no band. When you are not sure, say so.")
            ForEach(model.form.classQuestions(catalogue)) { q in
                ChoiceQuestion(
                    question: OnboardingText.question(q), choices: Choice.classes(q.classes, holds: q.holds),
                    selection: answer(q.key), columns: 3)
            }
            ForEach(model.form.stageSkills(catalogue)) { skill in
                stage(skill)
            }
            ForEach(model.form.mobilityChecks, id: \.self) { check in
                ChoiceQuestion(
                    question: OnboardingText.mobility(check), choices: Choice.list(OnboardingText.mobilityAnswers),
                    selection: mobility(check), columns: 3)
            }
            ChoiceQuestion(
                question: String(localized: "How sure are these numbers?"),
                choices: Choice.list(OnboardingText.dataConfidence), selection: $model.form.dataConfidence)
        }
    }

    private func stage(_ skill: OnboardingCatalogue.Skill) -> some View {
        VStack(alignment: .leading, spacing: 20) {
            ChoiceQuestion(
                question: stageQuestion(skill), choices: Choice.stages(skill), selection: stageLevel(skill.slug))
            if model.form.stages[skill.slug]?.namesLevel == true {
                ChoiceQuestion(
                    question: skill.holds ? String(localized: "For how long?") : String(localized: "How many?"),
                    choices: Choice.classes(skill.stageClasses, holds: skill.holds),
                    selection: stageClass(skill.slug), columns: 2)
            }
        }
    }

    private func stageQuestion(_ skill: OnboardingCatalogue.Skill) -> String {
        skill.holds
            ? String(localized: "Which stage of \(skill.name) can you hold cleanly?")
            : String(localized: "Which stage of \(skill.name) can you do cleanly?")
    }

    private func answer(_ key: String) -> Binding<String?> {
        Binding(get: { model.form.classes[key] }, set: { model.form.classes[key] = $0 })
    }

    private func mobility(_ check: String) -> Binding<String?> {
        Binding(get: { model.form.mobility[check] }, set: { model.form.mobility[check] = $0 })
    }

    private func stageLevel(_ skill: String) -> Binding<String?> {
        Binding(
            get: { model.form.stages[skill]?.level },
            set: { level in
                // A new level asks its class again.
                let kept = model.form.stages[skill]
                model.form.stages[skill] = level.map {
                    OnboardingForm.Stage(level: $0, answerClass: kept?.level == $0 ? kept?.answerClass : nil)
                }
            })
    }

    private func stageClass(_ skill: String) -> Binding<String?> {
        Binding(
            get: { model.form.stages[skill]?.answerClass },
            set: { model.form.stages[skill]?.answerClass = $0 })
    }
}

/// Block G: exertion, screening, the body map.
struct OnboardingHealthPage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            PageTitle(
                "Your health",
                detail: "Safety comes before the plan. Hefesto does not diagnose; where an answer calls for it, it recommends seeing a professional.")
            VStack(alignment: .leading, spacing: 12) {
                YesNoQuestion(question: catalogue.exertion.question, answer: $model.form.exertion)
                if model.form.exertion == true { Advice(text: catalogue.exertion.advice) }
            }
            if model.form.healthConsent == true {
                VStack(alignment: .leading, spacing: 20) {
                    ForEach(catalogue.screening) { q in
                        YesNoQuestion(question: q.text, answer: screening(q.id))
                    }
                }
            }
            VStack(alignment: .leading, spacing: 12) {
                QuestionText(
                    model.form.healthConsent == true
                        ? "Do you have complaints that affect your training? Did an injury stop your training in the last 12 months?"
                        : "Do you have complaints that affect your training?")
                ForEach(catalogue.regions) { region in
                    bodyMapRow(region)
                }
            }
        }
    }

    private func bodyMapRow(_ region: OnboardingCatalogue.Region) -> some View {
        let mark = model.form.bodyMap[region.id] ?? .init()
        return VStack(alignment: .leading, spacing: 8) {
            Text(verbatim: region.name).font(.bodyMedium).foregroundStyle(Palette.text)
            HStack(spacing: 8) {
                ChoiceButton(label: String(localized: "Now"), selected: mark.current) {
                    model.form.bodyMap[region.id, default: .init()].current.toggle()
                }
                if model.form.healthConsent == true {
                    ChoiceButton(label: String(localized: "In the last 12 months"), selected: mark.past12Months) {
                        model.form.bodyMap[region.id, default: .init()].past12Months.toggle()
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func screening(_ id: String) -> Binding<Bool?> {
        Binding(get: { model.form.screening[id] }, set: { model.form.screening[id] = $0 })
    }
}

/// A current complaint: its details and the region's warning signs.
struct OnboardingRegionPage: View {
    let regionId: String
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        if let region = catalogue.region(regionId) {
            content(region)
        }
    }

    private func content(_ region: OnboardingCatalogue.Region) -> some View {
        VStack(alignment: .leading, spacing: 28) {
            VStack(alignment: .leading, spacing: 6) {
                CapsLabel("Your complaint", isHeader: false)
                Text(verbatim: region.name)
                    .font(.displayTitle)
                    .foregroundStyle(Palette.text)
                    .accessibilityAddTraits(.isHeader)
            }
            ChoiceQuestion(
                question: String(localized: "How strong is it in daily life?"), choices: Choice.scale,
                selection: pain(\.painDaily), columns: 6)
            ChoiceQuestion(
                question: String(localized: "How strong is it during or right after training?"), choices: Choice.scale,
                selection: pain(\.painTraining), columns: 6)
            Text("0 is no pain, 10 the worst you can imagine.")
                .font(.fine)
                .foregroundStyle(Palette.textSecondary)
            ChoiceQuestion(
                question: String(localized: "How did it begin?"), choices: Choice.list(OnboardingText.onsets),
                selection: field(\.onset), columns: 2)
            ChoiceQuestion(
                question: String(localized: "Since when?"),
                choices: OnboardingForm.durationWeeks.map { Choice(id: "\($0)", label: OnboardingText.duration($0)) },
                selection: duration, columns: 2)
            ChoiceQuestion(
                question: String(localized: "Do you suspect a tear or another serious injury yourself?"),
                choices: Choice.list(OnboardingText.suspicions), selection: field(\.suspectedSerious), columns: 3)
            ChoiceQuestion(
                question: String(localized: "Has a professional looked at it?"),
                choices: Choice.list(OnboardingText.assessments), selection: field(\.assessment))
            restrictions
            flags(region)
        }
    }

    private var complaint: OnboardingForm.Complaint {
        model.form.complaints[regionId] ?? .init()
    }

    private var restrictions: some View {
        VStack(alignment: .leading, spacing: 10) {
            QuestionText("Did a professional restrict any movements? Optional.")
            ForEach(Choice.list(OnboardingText.restrictions)) { r in
                ChoiceButton(label: r.label, selected: complaint.restrictions.contains(r.id)) {
                    var c = complaint
                    if c.restrictions.contains(r.id) { c.restrictions.remove(r.id) } else { c.restrictions.insert(r.id) }
                    model.form.complaints[regionId] = c
                }
            }
        }
    }

    private func flags(_ region: OnboardingCatalogue.Region) -> some View {
        VStack(alignment: .leading, spacing: 20) {
            VStack(alignment: .leading, spacing: 6) {
                CapsLabel("Warning signs")
                Text("These questions look for signs that need a professional. They are not a diagnosis.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
            }
            ForEach(model.form.flags(for: region, catalogue, currentYear: model.currentYear)) { flag in
                VStack(alignment: .leading, spacing: 12) {
                    YesNoQuestion(question: flag.question, answer: flagAnswer(flag.id))
                    if model.form.redFlags[regionId]?[flag.id] == true {
                        if let followup = flag.followup {
                            YesNoQuestion(question: followup.question, answer: flagAnswer(followup.id))
                            Advice(text: model.form.redFlags[regionId]?[followup.id] == true ? followup.advice : flag.advice)
                        } else {
                            Advice(text: flag.advice)
                        }
                    }
                }
            }
        }
    }

    private func pain(_ path: WritableKeyPath<OnboardingForm.Complaint, Int?>) -> Binding<String?> {
        Binding(
            get: { complaint[keyPath: path].map { "\($0)" } },
            set: { value in
                var c = complaint
                c[keyPath: path] = value.flatMap { Int($0) }
                model.form.complaints[regionId] = c
            })
    }

    private func field(_ path: WritableKeyPath<OnboardingForm.Complaint, String?>) -> Binding<String?> {
        Binding(
            get: { complaint[keyPath: path] },
            set: { value in
                var c = complaint
                c[keyPath: path] = value
                model.form.complaints[regionId] = c
            })
    }

    private var duration: Binding<String?> {
        Binding(
            get: { complaint.durationWeeks.map { "\($0)" } },
            set: { value in
                var c = complaint
                c.durationWeeks = value.flatMap { Double($0) }
                model.form.complaints[regionId] = c
            })
    }

    private func flagAnswer(_ id: String) -> Binding<Bool?> {
        Binding(
            get: { model.form.redFlags[regionId]?[id] },
            set: { value in
                model.form.redFlags[regionId, default: [:]][id] = value
                // A follow-up is asked only after a yes.
                if value != true, let followup = catalogue.region(regionId)?.flags.first(where: { $0.id == id })?.followup {
                    model.form.redFlags[regionId]?[followup.id] = nil
                }
            })
    }
}

/// The last page: send, or answer the clarifications the server asked.
struct OnboardingFinishPage: View {
    let catalogue: OnboardingCatalogue
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            if let outcome = model.outcome, outcome.status == .needsAnswers {
                PageTitle(
                    "One more thing",
                    detail: "Some answers do not quite fit together. Without an answer here, the planner starts lower to be safe.")
                ForEach(outcome.questions) { q in
                    YesNoQuestion(question: q.text, answer: clarification(q.id))
                }
            } else {
                PageTitle(
                    "Ready",
                    detail: "Hefesto now plans your first week. The first sessions check your answers: the plan starts a little lower and learns from your log.")
            }
            Text(verbatim: catalogue.disclaimer).font(.fine).foregroundStyle(Palette.textSecondary)
        }
    }

    private func clarification(_ id: String) -> Binding<Bool?> {
        Binding(get: { model.form.clarifications[id] }, set: { model.form.clarifications[id] = $0 })
    }
}

/// What the onboarding led to: the first plan, or a stop and why.
struct OnboardingResultView: View {
    let outcome: OnboardingOutcome
    let onDone: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    if outcome.status == .blocked {
                        PageTitle("Training is paused", detail: nil)
                    } else {
                        PageTitle("Your plan is ready", detail: "Today shows your week. Rest days count towards the streak.")
                    }
                    notes(outcome.reasons)
                    notes(outcome.realism)
                    notes(outcome.hints)
                    Text(verbatim: outcome.disclaimer).font(.fine).foregroundStyle(Palette.textSecondary)
                }
                .padding(20)
            }
            Button(action: onDone) {
                if outcome.status == .blocked {
                    Label("Close", systemImage: "checkmark")
                } else {
                    Label("See my plan", systemImage: "checkmark")
                }
            }
            .buttonStyle(PrimaryButtonStyle())
            .padding(16)
        }
    }

    @ViewBuilder
    private func notes(_ list: [PlanNote]) -> some View {
        if !list.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                ForEach(Array(list.enumerated()), id: \.offset) { _, note in
                    Text(verbatim: note.text).font(.detailText).foregroundStyle(Palette.text)
                }
            }
            .card()
        }
    }
}

// MARK: - Building blocks

/// A page's title and what it is about.
struct PageTitle: View {
    let title: LocalizedStringKey
    let detail: LocalizedStringKey?

    init(_ title: LocalizedStringKey, detail: LocalizedStringKey?) {
        self.title = title
        self.detail = detail
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.displayTitle)
                .foregroundStyle(Palette.text)
                .accessibilityAddTraits(.isHeader)
            if let detail {
                Text(detail).font(.detailText).foregroundStyle(Palette.textSecondary)
            }
        }
    }
}

/// A question above its answers.
struct QuestionText: View {
    let text: Text

    init(_ key: LocalizedStringKey) { text = Text(key) }
    init(verbatim: String) { text = Text(verbatim: verbatim) }

    var body: some View {
        text
            .font(.rowTitle)
            .foregroundStyle(Palette.text)
            .fixedSize(horizontal: false, vertical: true)
    }
}

/// One answer among others.
struct Choice: Identifiable, Hashable {
    let id: String
    let label: String

    static func list(_ pairs: [(key: String, label: String)]) -> [Choice] {
        pairs.map { Choice(id: $0.key, label: $0.label) }
    }

    /// 0 to 10, for pain.
    static let scale = (0...10).map { Choice(id: "\($0)", label: "\($0)") }

    /// A class question's classes, and «don't know».
    static func classes(_ classes: [OnboardingCatalogue.AnswerClass], holds: Bool) -> [Choice] {
        var out = classes.map { Choice(id: $0.key, label: OnboardingText.answerClass($0, holds: holds)) }
        out.append(Choice(id: OnboardingForm.unknown, label: String(localized: "Don't know")))
        return out
    }

    /// A skill's stages: not yet, each level, and «don't know».
    static func stages(_ skill: OnboardingCatalogue.Skill) -> [Choice] {
        var out = [Choice(id: OnboardingForm.Stage.noLevel, label: String(localized: "Not yet"))]
        out += skill.levels.map { Choice(id: $0.slug, label: $0.name) }
        out.append(Choice(id: OnboardingForm.unknown, label: String(localized: "Don't know")))
        return out
    }
}

/// An answer as a large button; ember when chosen.
struct ChoiceButton: View {
    let label: String
    var detail: String?
    let selected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: label)
                    .font(.bodyMedium)
                    .foregroundStyle(selected ? Palette.emberInk : Palette.text)
                if let detail {
                    Text(verbatim: detail)
                        .font(.fine)
                        .foregroundStyle(selected ? Palette.emberInk : Palette.textSecondary)
                }
            }
            .multilineTextAlignment(.leading)
            .frame(maxWidth: .infinity, minHeight: 52, alignment: .leading)
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .background(selected ? Palette.ember : Palette.surface, in: .rect(cornerRadius: 12))
            .contentShape(.rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}

/// A question with its answers; a second tap on the chosen one clears it.
struct ChoiceQuestion: View {
    let question: String
    let choices: [Choice]
    @Binding var selection: String?
    var columns = 1

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            QuestionText(verbatim: question)
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: columns), spacing: 8) {
                ForEach(choices) { c in
                    ChoiceButton(label: c.label, selected: selection == c.id) {
                        selection = selection == c.id ? nil : c.id
                    }
                }
            }
        }
    }
}

/// A question answered yes or no. The knowledge base's questions are shown
/// verbatim.
struct YesNoQuestion: View {
    let question: String
    @Binding var answer: Bool?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            QuestionText(verbatim: question)
            HStack(spacing: 8) {
                ChoiceButton(label: String(localized: "No"), selected: answer == false) { answer = false }
                ChoiceButton(label: String(localized: "Yes"), selected: answer == true) { answer = true }
            }
        }
    }
}

/// What a yes means, in the knowledge base's words.
struct Advice: View {
    let text: String

    var body: some View {
        Label {
            Text(verbatim: text).font(.detailText).foregroundStyle(Palette.text)
        } icon: {
            Image(systemName: "exclamationmark.circle").foregroundStyle(Palette.ember)
        }
        .card(padding: 14)
    }
}

/// Calendar days as the API writes them, YYYY-MM-DD, in the athlete's
/// calendar.
enum DayFormat {
    static func string(_ date: Date) -> String {
        let c = Calendar.current.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }

    static func date(_ day: String) -> Date {
        PlanText.date(day)
    }
}
