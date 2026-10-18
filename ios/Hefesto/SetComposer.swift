import HefestoLogger
import HefestoStore
import SwiftUI

/// Builds one set: a single element, or several for a combo. Either way it
/// is saved as one set (CLAUDE.md: one code path for sets).
struct SetComposer: View {
    let db: AppDatabase
    let defaultRest: Int
    /// The planned set it performs, if any: its targets fill the fields.
    let planned: PlannedTarget?
    /// The elements, the planned rest and what the athlete had left.
    let onSave: ([ElementDraft], Int, SetReserve) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var elements: [Draft]
    @State private var rest: Int
    @State private var rir: Int?
    @State private var sirS: Int?
    @State private var picking: Int?

    /// The choices for reps in reserve, the log's 0–10.
    static let rirChoices = Array(0...10)
    /// The choices for a hold's reserve, within the log's 0–60 s.
    static let sirChoices = [0, 1, 2, 3, 4, 5, 6, 8, 10, 15, 20, 30, 45, 60]

    /// `exercise`, if given, is chosen for the first element already. A
    /// `planned` set opens on its targets and its planned rest.
    init(db: AppDatabase, defaultRest: Int, exercise: Exercise? = nil, planned: PlannedTarget? = nil,
         onSave: @escaping ([ElementDraft], Int, SetReserve) -> Void) {
        self.db = db
        self.defaultRest = defaultRest
        self.planned = planned
        self.onSave = onSave
        if let planned, !planned.drafts.isEmpty {
            _elements = State(initialValue: planned.drafts)
            _rest = State(initialValue: planned.rest ?? defaultRest)
        } else {
            var first = Draft()
            if let exercise { first.choose(exercise) }
            _elements = State(initialValue: [first])
            _rest = State(initialValue: defaultRest)
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                if let planned { targetSection(planned) }
                ForEach($elements) { $draft in
                    let index = elements.firstIndex { $0.id == draft.id } ?? 0
                    Section {
                        Button {
                            picking = index
                        } label: {
                            HStack {
                                Text(verbatim: draft.exercise?.name ?? String(localized: "Choose an exercise"))
                                    .font(.rowTitle)
                                    .foregroundStyle(draft.exercise == nil ? Palette.textSecondary : Palette.text)
                                Spacer()
                                Image(systemName: "chevron.right").foregroundStyle(Palette.textTertiary)
                            }
                            .frame(minHeight: 44)
                        }
                        .themedRow()
                        if draft.exercise != nil { ElementFields(draft: $draft) }
                    } header: {
                        CapsLabel(elements.count > 1 ? LocalizedStringKey("Combo, part \(index + 1)") : "Set")
                    } footer: {
                        if elements.count > 1 {
                            Button("Remove this part", role: .destructive) {
                                elements.removeAll { $0.id == draft.id }
                            }
                            .font(.detailText)
                        }
                    }
                }

                // A planned set is done as planned: one item of the plan.
                if planned == nil {
                    Section {
                        Button("Add an exercise to make a combo", systemImage: "plus.square.on.square") {
                            elements.append(Draft())
                        }
                        .font(.bodyMedium)
                        .foregroundStyle(Palette.ember)
                        .frame(minHeight: 44)
                        .themedRow()
                    }
                }

                if asksRIR || asksSIR {
                    Section {
                        // A group so the theme reaches every row it makes.
                        Group {
                            if asksRIR {
                                Picker("Reps left", selection: $rir) {
                                    Text("Not rated").tag(Int?.none)
                                    ForEach(Self.rirChoices, id: \.self) { Text($0, format: .number).tag(Int?.some($0)) }
                                }
                                .themedPicker()
                            }
                            if asksSIR {
                                Picker("Seconds left", selection: $sirS) {
                                    Text("Not rated").tag(Int?.none)
                                    ForEach(Self.sirChoices, id: \.self) { Text("\($0) s").tag(Int?.some($0)) }
                                }
                                .themedPicker()
                            }
                        }
                        .themedRow()
                    } header: {
                        CapsLabel("Reserve")
                    } footer: {
                        VStack(alignment: .leading, spacing: 6) {
                            if planned?.calibration == true {
                                Text("This set calibrates your plan: what you had left shows it where you stand.")
                                    .foregroundStyle(Palette.text)
                            }
                            if asksRIR { Text("How many more clean reps you could have done.") }
                            if asksSIR { Text("How much longer you could have held with clean form.") }
                        }
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)
                    }
                }

                Section {
                    Stepper(value: $rest, in: 0...600, step: 15) {
                        Text(Duration.seconds(rest).formatted(.time(pattern: .minuteSecond)))
                            .font(.metricSmall)
                            .monospacedDigit()
                            .foregroundStyle(Palette.text)
                    }
                    .themedRow()
                } header: {
                    CapsLabel("Rest after")
                }
            }
            .themedScreen()
            .navigationTitle(planned == nil ? Text("Log a set") : Text("Planned set"))
            .navigationBarTitleDisplayMode(.inline)
            .font(.bodyText)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        onSave(
                            elements.compactMap(\.element), rest,
                            SetReserve(rir: asksRIR ? rir : nil, sirS: asksSIR ? sirS : nil))
                        dismiss()
                    }
                    .fontWeight(.semibold)
                    .disabled(!elements.allSatisfy { $0.element != nil })
                }
            }
            .sheet(item: Binding(get: { picking.map(Picking.init) }, set: { picking = $0?.index })) { p in
                ExercisePicker(db: db) { exercise in
                    if elements.indices.contains(p.index) { elements[p.index].choose(exercise) }
                    picking = nil
                }
            }
        }
    }

    /// What the plan asked for, above the fields that start on it.
    private func targetSection(_ planned: PlannedTarget) -> some View {
        Section {
            VStack(alignment: .leading, spacing: 6) {
                if let reserve = planned.reserveTarget {
                    Text(verbatim: reserve).font(.bodyMedium).foregroundStyle(Palette.text)
                }
                if planned.calibration {
                    Chip("Calibration")
                }
                Text("Enter what you did; the target is filled in.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
            }
            .padding(.vertical, 4)
            .themedRow()
        } header: {
            CapsLabel("Target")
        }
    }

    private var asksRIR: Bool { asksReserve(of: "reps") }
    private var asksSIR: Bool { asksReserve(of: "hold_seconds") }

    /// A reserve belongs to the set, so it is asked only when exactly one
    /// element has the measure: with two, it would not say which one. An
    /// element taken to failure had nothing left.
    private func asksReserve(of measure: String) -> Bool {
        let matching = elements.filter { $0.exercise != nil && $0.measure == measure }
        return matching.count == 1 && !matching[0].failed
    }
}

/// A planned set to perform (ADR 0022): what the plan asked for.
struct PlannedTarget {
    let setId: String
    /// The planned elements as drafts, their targets entered.
    let drafts: [Draft]
    let rest: Int?
    /// What should be left: "2 reps in reserve".
    let reserveTarget: String?
    /// The set calibrates the plan's estimate, so its reserve matters.
    let calibration: Bool
}

/// What the athlete had left after a set: reps for a rep element, seconds
/// for a hold. Either may be unrated.
struct SetReserve {
    var rir: Int?
    var sirS: Int?
}

private struct Picking: Identifiable {
    let index: Int
    var id: Int { index }
}

/// The athlete's entry for one element, before it is a draft for the logger.
struct Draft: Identifiable {
    let id = UUID()
    var exercise: Exercise?
    var measure = "reps"
    var reps = 5
    var holdSeconds = 20
    var distanceM = 20
    var loadKg: Double = 0
    var assistance = "none"
    var assistKg: Double = 0
    var formQuality: Int?
    var failed = false

    mutating func choose(_ e: Exercise) {
        exercise = e
        measure = e.defaultMeasure
    }

    /// The draft for the logger, once an exercise is chosen.
    var element: ElementDraft? {
        guard let exercise else { return nil }
        return ElementDraft(
            exerciseId: exercise.id, measure: measure,
            reps: measure == "reps" ? reps : nil,
            holdSeconds: measure == "hold_seconds" ? Double(holdSeconds) : nil,
            distanceM: measure == "distance_m" ? Double(distanceM) : nil,
            loadKg: loadKg,
            assistance: assistance == "none" ? nil : Assistance(type: assistance, estimatedAssistKg: assistKg > 0 ? assistKg : nil),
            formQuality: formQuality, failed: failed)
    }
}

extension Draft {
    /// A planned element's targets, ready to change into what was done.
    init(target e: SetElement, exercise: Exercise?) {
        self.init()
        self.exercise = exercise
        measure = e.measure
        if let r = e.reps { reps = r }
        if let h = e.holdSeconds { holdSeconds = Int(h.rounded()) }
        if let d = e.distanceM { distanceM = Int(d.rounded()) }
        loadKg = e.loadKg
        if let a = e.assistance, AssistanceKind.all.contains(a.type) {
            assistance = a.type
            assistKg = a.estimatedAssistKg ?? 0
        }
    }
}

struct ElementFields: View {
    @Binding var draft: Draft

    var body: some View {
        // A group so the theme reaches every row it makes.
        Group { fields }.themedRow()
    }

    @ViewBuilder
    private var fields: some View {
        switch draft.measure {
        case "reps":
            Stepper(value: $draft.reps, in: 0...500) {
                Text("\(draft.reps) reps").font(.metricSmall).monospacedDigit()
            }
            .frame(minHeight: 56)
        case "hold_seconds":
            Stepper(value: $draft.holdSeconds, in: 0...3600, step: 5) {
                Text("\(draft.holdSeconds) s hold").font(.metricSmall).monospacedDigit()
            }
            .frame(minHeight: 56)
        case "distance_m":
            Stepper(value: $draft.distanceM, in: 0...10000, step: 5) {
                Text("\(draft.distanceM) m").font(.metricSmall).monospacedDigit()
            }
            .frame(minHeight: 56)
        default:
            EmptyView()
        }

        Stepper(value: $draft.loadKg, in: 0...300, step: 2.5) {
            LabeledContent("Added load") {
                Text("\(draft.loadKg.formatted(.number.precision(.fractionLength(0...1)))) kg")
                    .monospacedDigit()
                    .foregroundStyle(Palette.textSecondary)
            }
        }

        Picker("Assistance", selection: $draft.assistance) {
            ForEach(AssistanceKind.all, id: \.self) { Text(AssistanceKind.label($0)).tag($0) }
        }
        .themedPicker()
        if draft.assistance != "none" {
            LabeledContent("Estimated assist") {
                TextField("kg", value: $draft.assistKg, format: .number)
                    .keyboardType(.decimalPad)
                    .multilineTextAlignment(.trailing)
            }
        }

        Picker("Form", selection: $draft.formQuality) {
            Text("Not rated").tag(Int?.none)
            ForEach(1...5, id: \.self) { Text(FormQuality.label($0)).tag(Int?.some($0)) }
        }
        .themedPicker()
        Toggle("To failure", isOn: $draft.failed)
    }
}

enum AssistanceKind {
    /// Band assistance names one of the athlete's bands; it arrives with the
    /// bands screen. The other kinds need nothing more.
    static let all = ["none", "partner", "machine", "incline", "counterweight", "foot_support"]

    static func label(_ kind: String) -> String {
        switch kind {
        case "none": String(localized: "None")
        case "band": String(localized: "Band")
        case "partner": String(localized: "Partner")
        case "machine": String(localized: "Machine")
        case "incline": String(localized: "Incline")
        case "counterweight": String(localized: "Counterweight")
        case "foot_support": String(localized: "Foot support")
        default: kind
        }
    }
}

enum FormQuality {
    static func label(_ q: Int) -> String {
        switch q {
        case 1: String(localized: "1 · Poor")
        case 2: String(localized: "2 · Rough")
        case 3: String(localized: "3 · Solid")
        case 4: String(localized: "4 · Clean")
        default: String(localized: "5 · Textbook")
        }
    }
}

/// The cached catalogue, searched by name.
struct ExercisePicker: View {
    let db: AppDatabase
    let onPick: (Exercise) -> Void
    @State private var query = ""
    @State private var results: [Exercise] = []

    var body: some View {
        NavigationStack {
            List(results) { e in
                Button {
                    onPick(e)
                } label: {
                    VStack(alignment: .leading) {
                        Text(verbatim: e.name).font(.rowTitle).foregroundStyle(Palette.text)
                        Text(verbatim: FamilyText.label(e.family)).font(.fine).foregroundStyle(Palette.textSecondary)
                    }
                    .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .themedRow()
            }
            .themedScreen()
            .overlay {
                if results.isEmpty {
                    ContentUnavailableView(
                        query.isEmpty ? LocalizedStringKey("No exercises yet") : "No match",
                        systemImage: "magnifyingglass",
                        description: Text(query.isEmpty ? LocalizedStringKey("The catalogue loads with the first sync.") : "Try another name."))
                }
            }
            .searchable(text: $query, placement: .navigationBarDrawer(displayMode: .always))
            .navigationTitle("Exercise")
            .navigationBarTitleDisplayMode(.inline)
            .task(id: query) {
                do { for try await r in db.observeExercises(matching: query) { results = r } } catch {}
            }
        }
    }
}
