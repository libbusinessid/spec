# Prompt d’implémentation — moteur Kotlin EntID

## Mission

Implémente intégralement le dépôt Kotlin de EntID après lecture complète de
`engine.md`. Avant de coder, exige et lis aussi `rules.proto`, `conformance.proto`,
`ir.md`, `features.md`, le manifeste, `SHA256SUMS`, l’attestation et les deux BINPB
de la même release. Si un élément manque ou se contredit, arrête-toi et fais corriger
`spec` ; ne déduis jamais une sémantique du code Go de référence. La cible initiale
est Android/JVM, mais le cœur ne doit dépendre d’aucune API Android afin de rester
utilisable dans tout projet JVM et de préserver une voie
future vers Java ou Kotlin Multiplatform.

Le moteur doit être idiomatique Kotlin, immuable, sûr en concurrence, fortement
testé et publiable sur Maven Central. Ne traduis pas mécaniquement un autre moteur.

## Plateforme et build

- Gradle Wrapper commité et vérifié ;
- Gradle Kotlin DSL uniquement ;
- plugin Kotlin/JVM stable ;
- JDK toolchain verrouillé, avec bytecode cible documenté et compatible Android ;
- aucune dépendance au SDK Android dans le module cœur ;
- Protobuf officiel Kotlin/Java **dans le module générateur uniquement** ; la
  bibliothèque publiée n'a aucune dépendance Protobuf, puisqu'elle ne décode rien ;
- versions minimales de Kotlin, JDK et Android consommateur documentées ;
- publication Maven avec sources, Javadoc/Dokka, POM, checksums et signature.

Structure recommandée :

```text
.
├── build.gradle.kts
├── settings.gradle.kts
├── gradle/libs.versions.toml
├── gradlew
├── src/main/kotlin/org/entid/
│   ├── api/
│   ├── domain/
│   ├── runtime/          # primitives que le code généré appelle
│   ├── generated/        # code émis depuis le bundle, jamais édité
│   └── internal/
├── generator/            # le générateur : lit le bundle, valide, émet
│   └── src/main/kotlin/
├── src/test/kotlin/
└── rules.lock
```

Le générateur et la bibliothèque sont deux modules Gradle. Le décodage Protobuf, les
vingt-cinq contrôles de chargement et la connaissance des opcodes vivent dans le
générateur ; la bibliothèque publiée ne contient que le code émis, les primitives
qu'il appelle et l'API.

Aucun `.binpb` n'est une ressource du jar. Le bundle est une entrée du générateur, lue
à la construction — l'embarquer ferait porter à chaque appelant une donnée que le code
généré rend inutile, et supposerait un décodeur pour la lire.

Si une future structure multiplateforme est préparée, ne prétends pas supporter KMP
tant que Protobuf et tous les targets ne passent pas la conformité. La V1 annonce
clairement JVM/Android.

## API Kotlin attendue

Concevoir une API proche de :

```kotlin
public data class IdentifierInput(
    val kind: IdentifierKind,
    val value: String,
    val countryCode: String? = null,
)

public class EntIdEngine private constructor(...) {
    public fun canonicalize(
        input: IdentifierInput,
        options: ValidationOptions = ValidationOptions(),
    ): CanonicalizationResult

    public fun validate(
        input: IdentifierInput,
        options: ValidationOptions = ValidationOptions(),
    ): ValidationReport

    public fun validateFormat(
        input: IdentifierInput,
        options: ValidationOptions = ValidationOptions(),
    ): ValidationReport

    public fun validateChecksum(
        input: IdentifierInput,
        options: ValidationOptions = ValidationOptions(),
    ): ValidationReport

    public companion object {
        public fun default(): EntIdEngine
    }
}
```

Il n’y a pas de fabrique acceptant un bundle en octets à l’exécution : une version
antérieure de ce document en déclarait une, ce qui imposait de porter le validateur
complet et la machine d’exécution de l’IR chez chaque appelant — c’est-à-dire un
interpréteur, que `engine.md` section 1.2 interdit. Un jeu de règles personnalisé
passe par le générateur, à la construction.

Les entrées métier invalides sont des rapports, pas des exceptions.

Principes :

- `data class` immuables pour les valeurs ;
- `enum class` ou sealed hierarchy stable pour statuts/raisons ;
- `IdentifierKind` est une value class chaîne extensible avec constantes connues,
  pas un `enum class` fermé ;
- aucune collection mutable exposée ;
- copies défensives de `ByteArray` ;
- aucune fuite des classes Protobuf ;
- nullabilité explicite, aucun `!!` dans le cœur ;
- API utilisable naturellement depuis Java lorsque cela ne dégrade pas Kotlin ;
- noms JSON communs si `kotlinx.serialization` est proposé, sans rendre cette
  dépendance obligatoire sauf décision documentée ;
- pas de `isValid` ambigu sur le rapport complet.

L’instance par défaut doit être initialisée de façon lazy et thread-safe. Après
construction, le moteur est totalement immuable.

### Registre : rien à livrer en V1

Ce moteur n'expose aucun type de registre. `engine.md` section 10 diffère la
consultation d'un registre distant à une version ultérieure, et un moteur qui n'en
porte pas est pleinement conforme.

Trois choses à ne pas faire, parce qu'elles fermeraient la porte :

- n'exposez aucun type de registre, même marqué expérimental : un type public est un
  engagement que SemVer fige ;
- ne rendez aucune méthode de validation asynchrone « au cas où » — la validation
  locale reste synchrone définitivement ;
- ne mettez aucune dépendance HTTP dans le paquet du cœur.

Une consultation de registre porte un jeton d'API : elle ne devra jamais être possible
depuis un navigateur, et vivra donc dans un module séparé, chargé côté serveur
uniquement.


## Protobuf, dans le générateur

Tout ce qui suit concerne le module générateur. La bibliothèque publiée ne décode
rien, ne dépend d'aucun runtime Protobuf, et ne charge aucune ressource.

- Générer les classes Protobuf dans le module générateur, jamais dans le module publié.
- Utiliser le plugin Gradle Protobuf avec versions verrouillées.
- Vérifier si `protobuf-kotlin-lite`/Java lite satisfait unknown fields, oneof et
  limites nécessaires ; couvrir le choix par tests. Les champs inconnus doivent rester
  lisibles : le contrôle 5 en dépend, et un runtime qui les jette rend un écart de
  version indétectable.
- Ajouter tâches `generateProto` et `checkGenerated`.
- Le générateur lit le bundle depuis un chemin, pas depuis le classpath : c'est une
  entrée de build, pas une ressource.
- Ne pas utiliser les maps/messages générés comme structures runtime mutables.

Vérifier par un test que le jar publié ne contient aucune classe Protobuf ni aucun
`.binpb` : c'est ce qui distingue un moteur généré d'un interpréteur, et une
dépendance Gradle mal portée le trahit silencieusement.

## Style et qualité Kotlin

- Kotlin explicit API mode activé ;
- warnings as errors ;
- ktlint et Detekt avec configurations versionnées ;
- format officiel Kotlin ;
- pas de wildcard imports ;
- pas de `!!`, casts non sûrs ou exceptions génériques dans le cœur ;
- sealed types exhaustifs et `when` exhaustifs ;
- vérifier les conversions `UInt`/`Int`/`Long` et indices ;
- ne pas utiliser `String.length` comme nombre de points de code sans abstraction
  conforme à `engine.md` ;
- ne pas utiliser `uppercase()` dépendant de locale ; utiliser ASCII explicite ;
- ne pas dépendre de `java.util.regex` pour les règles ;
- pas de mutable global ou object state non synchronisé ;
- KDoc sur toutes les APIs publiques ;
- complexité contenue et suppressions Detekt locales/commentées.

Ajouter Kotlin Binary Compatibility Validator ou outil équivalent pour suivre l’API
publique, ainsi que Dokka pour la documentation.

## Tests Kotlin obligatoires

- `kotlin.test`/JUnit 5 avec convention cohérente ;
- tests paramétrés pour toutes les opérations ;
- conformité complète : le corpus pilote le code émis, pas un décodeur ;

Le runner de conformité vient de `spec`, jamais de ce dépôt. Il s'exécute épinglé au
commit que `rules.lock` enregistre sous `source_commit` :

```bash
go run github.com/entid-org/spec/cmd/conformance-runner@<source_commit> \
  -corpus spec/entid-conformance.binpb -- ./gradlew -q runTestee
```

N'écris pas de comparateur : un moteur qui juge lui-même ses propres résultats peut
se déclarer conforme en comparant trop faiblement. Ce que tu écris, c'est le testee,
et les tests qui prouvent qu'il ne triche pas.

`invalid_encoding` ne peut pas être un cas de conformité — `ir.md` section 5 étape 1
dit pourquoi. Épingle-le par un test natif portant une `String` contenant une demi-paire
de substitution non appariée, que le type `String` de Kotlin admet.

- bundles invalides et limites, contre le générateur ;
- packaging : que le jar publié ne porte ni `.binpb` ni classe Protobuf ;
- tests multithread avec executors/coroutines si disponibles ;
- tests de compatibilité Android dans un projet consommateur minimal ;
- tests de l’API depuis Java pour les points importants ;
- property-based testing avec Kotest Property ou outil maintenu ;
- fuzzing JVM via Jazzer ou outil équivalent pour décodeur et runtime ;
- benchmarks JMH pour chargement et validation.

Utiliser Kover ou JaCoCo avec :

- lignes/instructions ≥ 95 % ;
- branches ≥ 90 % ;
- 100 % dispatch IR et refus version/feature ;
- exclusions limitées au code Protobuf généré ;
- rapport XML/HTML conservé en CI.

Mutation testing via Pitest avec support Kotlin compatible, ou outil équivalent,
ciblé sur runtime/checksums ; objectif recommandé ≥ 80 % et analyse des survivants.

## CI

GitHub Actions doit exécuter :

- Gradle Wrapper validation ;
- build sur JDK minimal et actuel ;
- ktlint, Detekt, explicit API et warnings-as-errors ;
- tests unitaires + conformité ;
- couverture et seuils ;
- fuzz smoke ;
- API binary compatibility ;
- `checkGenerated` ;
- audit de dépendances ;
- assemblage JAR/AAR si annoncé ;
- test consommateur JVM et test Android minimal ;
- publication Maven en dry-run sur PR pertinente.

CI planifiée : Jazzer long, mutation testing, benchmarks et versions de toolchain.

## Documentation et publication

README : installation Gradle/Maven, exemple Kotlin, exemple Java minimal, format seul,
checksum unsupported, jeu de règles personnalisé passant par le générateur et
versions. Indiquer
explicitement que JVM/Android est le support V1 et que KMP n’est pas annoncé.

Publier sources et Dokka, signer les artefacts et suivre SemVer. Ajouter SECURITY,
CONTRIBUTING et changelog.

## Interdictions spécifiques

- aucune classe Android dans le cœur ;
- aucun `!!` ou global mutable ;
- aucun type Protobuf public ;
- aucune coroutine ou client réseau dans `validate` ;
- aucune regex recréant les règles ;
- aucune prétention KMP sans conformité sur chaque target ;
- aucune copie ligne par ligne du moteur Go/Swift ;
- aucun cas de conformité désactivé ;
- aucun type de registre, même expérimental.

## Livrables et définition de terminé

Livrer build Gradle reproductible, générateur, bibliothèque, API, moteur, tests, property tests, fuzzing, benchmarks, CI, documentation Dokka et
publication Maven. Tous les critères `engine.md`, lint, conformité, couverture,
compatibilité JVM/Android et test consommateur doivent être verts, sans TODO V1 ni
warning.
