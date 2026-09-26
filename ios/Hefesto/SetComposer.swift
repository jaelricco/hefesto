import HefestoLogger
import HefestoStore
import SwiftUI

/// Builds one set: a single element, or several for a combo. Either way it
/// is saved as one set (CLAUDE.md: one code path for sets).
struct SetComposer: View {
    let db: AppDatabase
    let defaultRest: Int
    let onSave: ([ElementDraft], Int) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var elements: [Draft] = [Draft()]
    @State private var rest: Int
    @State private var picking: Int?

    init(db: AppDatabase, defaultRest: Int, onSave: @escaping ([ElementDraft], Int) -> Void) {
        self.db = db
        self.defaultRest = defaultRest
        self.onSave = onSave
        _rest = State(initialValue: defaultRest)
    }

    var body: some View {
        NavigationStack {
            Form {
                ForEach($elements) { $draft in
                    let index = elements.firstIndex { $0.id == draft.id } ?? 0
                    Section {
                        Button {
                            picking = index
                        } label: {
                            HStack {
                                Text(verbatim: draft.exercise?.name ?? String(localized: "Choose an exercise"))
                                    .foregroundStyle(draft.exercise == nil ? .secondary : .primary)
                                Spacer()
                                Image(systemName: "chevron.right").foregroundStyle(.tertiary)
                            }
                            .frame(minHeight: 44)
                        }
                        if draft.exercise != nil { ElementFields(draft: $draft) }
                    } header: {
                        Text(elements.count > 1 ? LocalizedStringKey("Combo, part \(index + 1)") : "Set")
                    } footer: {
                        if elements.count > 1 {
                            Button("Remove this part", role: .destructive) {
                                elements.removeAll { $0.id == draft.id }
                            }
                        }
                    }
                }

                Section {
                    Button("Add an exercise to make a combo", systemImage: "plus.square.on.square") {
                        elements.append(Draft())
                    }
                    .frame(minHeight: 44)
                }

                Section("Rest after") {
                    Stepper(value: $rest, in: 0...600, step: 15) {
                        Text(Duration.seconds(rest).formatted(.time(pattern: .minuteSecond)))
                            .font(.title3.monospacedDigit())
                    }
                }
            }
            .navigationTitle("Log a set")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        onSave(elements.compactMap(\.element), rest)
                        dismiss()
                    }
                    .bold()
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

struct ElementFields: View {
    @Binding var draft: Draft

    var body: some View {
        switch draft.measure {
        case "reps":
            Stepper(value: $draft.reps, in: 0...500) {
                Text("\(draft.reps) reps").font(.title2.monospacedDigit().bold())
            }
            .frame(minHeight: 56)
        case "hold_seconds":
            Stepper(value: $draft.holdSeconds, in: 0...3600, step: 5) {
                Text("\(draft.holdSeconds) s hold").font(.title2.monospacedDigit().bold())
            }
            .frame(minHeight: 56)
        case "distance_m":
            Stepper(value: $draft.distanceM, in: 0...10000, step: 5) {
                Text("\(draft.distanceM) m").font(.title2.monospacedDigit().bold())
            }
            .frame(minHeight: 56)
        default:
            EmptyView()
        }

        Stepper(value: $draft.loadKg, in: 0...300, step: 2.5) {
            LabeledContent("Added load") {
                Text("\(draft.loadKg.formatted(.number.precision(.fractionLength(0...1)))) kg").monospacedDigit()
            }
        }

        Picker("Assistance", selection: $draft.assistance) {
            ForEach(AssistanceKind.all, id: \.self) { Text(AssistanceKind.label($0)).tag($0) }
        }
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
                        Text(verbatim: e.name).font(.headline)
                        Text(verbatim: e.family).font(.caption).foregroundStyle(.secondary)
                    }
                    .frame(minHeight: 44)
                }
                .foregroundStyle(.primary)
            }
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
