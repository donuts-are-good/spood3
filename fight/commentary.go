package fight

import (
	"fmt"
	"math/rand"
	"spoodblort/database"
	"spoodblort/utils"
)

// Announcer represents a fictitious commentator
type Announcer struct {
	Name  string
	Style string // personality/style of commentary
}

// LiveAction represents a single moment in the fight
type LiveAction struct {
	Type       string `json:"type"`        // "damage", "round", "death", "special"
	Action     string `json:"action"`      // The combat description
	Damage     int    `json:"damage"`      // Damage dealt (if applicable)
	Attacker   string `json:"attacker"`    // Fighter who dealt damage
	Victim     string `json:"victim"`      // Fighter who took damage
	Commentary string `json:"commentary"`  // Announcer comment
	Announcer  string `json:"announcer"`   // Which announcer said it
	Health1    int    `json:"health1"`     // Fighter 1 current health
	Health2    int    `json:"health2"`     // Fighter 2 current health
	Round      int    `json:"round"`       // Current round number
	TickNumber int    `json:"tick_number"` // Current tick
}

var announcers = []Announcer{
	{"Chud Puncherson", "enthusiastic"},
	{"Dr. Mayhem PhD", "scientific"},
	{"\"Screaming\" Sally Bloodworth", "intense"},
	{"THE COMMISSIONER", "mysterious"},
}

var combatActions = []string{
	"BONE-CRUSHING HAYMAKER",
	"DEVASTATING ELBOW DROP FROM THE TOP ROPE",
	"BRUTAL KNEE TO THE SOLAR PLEXUS",
	"VICIOUS UPPERCUT SENDS TEETH FLYING",
	"CATASTROPHIC BODY SLAM SHAKES THE ARENA",
	"LIGHTNING-FAST JAB TO THE TEMPLE",
	"MERCILESS LIVER PUNCH",
	"EARTH-SHATTERING ROUNDHOUSE KICK",
	"SAVAGE HEADBUTT TO THE NOSE",
	"SPINE-TINGLING CHOKEHOLD",
	"APOCALYPTIC SPINNING BACKFIST",
	"SOUL-CRUSHING KNEE DROP",
	"REALITY-BENDING SUPLEX",
	"DIMENSION-SPLITTING CLOTHESLINE",
	"CHAOS-INDUCING PILE DRIVER",
	"THE COMMISSIONER'S FIST",
	"THE PURPLE NURPLE",
	"EARTH SHATTERING KICK",
	"SKULL-FRACTURING HAMMER FIST",
	"RIBCAGE-SHATTERING KNEE STRIKE",
	"CARTILAGE-PULVERIZING ELBOW SLAM",
	"INTESTINE-REARRANGING BODY BLOW",
	"FACIAL-RECONSTRUCTION UPPERCUT",
	"SPLEEN-LIQUIDATING HOOK",
	"VERTEBRAE-SNAPPING GERMAN SUPLEX",
	"ORGAN-SHUFFLING POWERBOMB",
	"KIDNEY-RUPTURING SIDE KICK",
	"TRACHEA-CRUSHING THROAT PUNCH",
	"STERNUM-CRACKING DOUBLE AXE HANDLE",
	"FEMUR-SPLITTING LEG DROP",
	"JAW-DISLOCATING HAYMAKER",
	"CRANIUM-DENTING SLEDGEHAMMER BLOW",
	"PELVIS-GRINDING HIP TOSS",
	"SHOULDER-SEPARATING CLOTHESLINE",
	"ANKLE-TWISTING DRAGON SCREW",
	"WRIST-SNAPPING ARM BAR",
	"CLAVICLE-SHATTERING SHOULDER TACKLE",
	"METACARPAL-CRUSHING KNUCKLE SANDWICH",
	"LUMBAR-DESTROYING BACKBREAKER",
	"PATELLA-PULVERIZING KNEE SMASH",
	"OCCIPITAL-OBLITERATING RABBIT PUNCH",
	"MANDIBLE-MANGLING JAW BREAKER",
	"TIBIA-SPLITTING SHIN KICK",
	"ULNA-FRACTURING FOREARM SMASH",
	"SCAPULA-CRUSHING SHOULDER BOMB",
	"TEMPORAL-TRAUMATIZING TEMPLE STRIKE",
	"CERVICAL-COMPRESSING NECK CRANK",
	"THORACIC-THRASHING CHEST BLOW",
	"SACRAL-SMASHING TAILBONE DROP",
	"PHALANGE-PULPING FINGER TWIST",
	"MAXILLA-MAULING FACE PLANT",
	"ORBITAL-OBLITERATING EYE GOUGE",
	"NASAL-DEMOLISHING NOSE BREAKER",
	"ZYGOMATIC-ZAPPING CHEEKBONE CRUSH",
	"HYOID-HAMMERING THROAT CHOP",
	"MASTOID-MASHING EAR CLAP",
	"FRONTAL-FRACTURING FOREHEAD BASH",
	"PARIETAL-POUNDING SKULL CRACK",
	"OCCIPITAL-ANNIHILATING HEAD SLAM",
	"MANDIBULAR-MUTILATING CHIN CHECK",
	"MAXILLARY-MANGLING UPPER JAW SMASH",
	"MOLAR-MASHING TOOTH CHIPPER",
	"INCISOR-OBLITERATING DENTAL DESTRUCTION",
	"BICUSPID-BREAKING BITE BLOCKER",
	"WISDOM-TOOTH-WRECKING MOUTH BOMB",
	"CANINE-CRUSHING FANG FRACTURE",
	"PREMOLAR-PULVERIZING GRIN GRINDER",
	"ENAMEL-ERASING SMILE SMASHER",
	"PERIODONTAL-PUNISHING GUM GRINDER",
	"ROOT-CANAL-RUPTURING CAVITY CREATOR",
	"ORTHODONTIC-OBLITERATING BRACE BREAKER",
	"GINGIVITIS-GENERATING GAP MAKER",
	"PLAQUE-PRODUCING TARTAR TERROR",
	"FLUORIDE-FRACTURING FILLING DESTROYER",
	"CROWN-CRACKING DENTAL DEVASTATION",
	"BRIDGE-BREAKING BITE BUSTER",
	"DENTURE-DEMOLISHING MOUTH MAYHEM",
	"RETAINER-RUPTURING JAW JAMMER",
	"VENEER-VANISHING SMILE SLAUGHTER",
	"ABSCESS-AMPLIFYING TOOTH TRAUMA",
	"PULP-PUMMELING NERVE NIGHTMARE",
	"DENTIN-DESTROYING CALCIUM CRUSHER",
	"SALIVA-STOPPING SPIT SPLITTER",
	"TONGUE-TWISTING TASTE BUD TERROR",
	"UVULA-UPROOTING THROAT THRASHER",
	"EPIGLOTTIS-ERADICATING SWALLOW STOPPER",
	"LARYNX-LIQUIDATING VOICE VOID",
	"PHARYNX-POUNDING GULLET GRINDER",
	"ESOPHAGUS-ELIMINATING TUBE TRASHER",
	"TONSIL-TERRORIZING THROAT THUMP",
}

var chadComments = []string{
	"HOLY MOLY! Did you see that molecular realignment?!",
	"That's gonna leave a mark on their existential dread!",
	"I'VE NEVER SEEN VIOLENCE THIS BEAUTIFUL!",
	"That fighter just got sent to the SHADOW REALM!",
	"SWEET MOTHER OF CHAOS! What a hit!",
	"Their ancestors felt that one from the afterlife!",
	"That's some premium-grade violence right there!",
	"OH MY GOODNESS GRACIOUS! The carnage is magnificent!",
	"WOWZA! That's what I call quality entertainment!",
	"GOLLY GEE! Someone's gonna need a new molecular structure!",
	"HOOBOY! That punch just violated several laws of physics!",
	"GREAT GOOGLY MOOGLY! The chaos energy is off the charts!",
	"YOWZA! I think I just witnessed interdimensional violence!",
	"JEEPERS CREEPERS! That's some premium brutality right there!",
	"HOLY CANNOLI! The existential dread levels are SPIKING!",
	"SWEET BABY MOSES! That fighter just got discombobulated!",
	"GOOD GOLLY MISS MOLLY! The violence is absolutely pristine!",
	"WOW WEE! Someone's getting their atoms rearranged!",
	"CHEESE AND CRACKERS! That's championship-level destruction!",
	"GOSH DIDDLY DANG! The molecular carnage is SUBLIME!",
	"HOLY MACKEREL! That's what I call a spine-tingling experience!",
	"JUMPING JACKRABBITS! Someone just got reality-checked!",
	"GREAT SCOTT! That fighter's anatomy just got redecorated!",
	"SWEET SALLY SUNSHINE! The violence is so wholesome!",
	"GOLLY WILLIKERS! That's some A-grade bone-crushing action!",
	"HOLY SMOKES! Their DNA just got a complete makeover!",
	"JIMINY CHRISTMAS! What a delightfully brutal exchange!",
	"GEE WHIZ! Someone's nervous system just took a vacation!",
	"SWEET PICKLED PEPPERS! That's textbook cranium crushing!",
	"GOOD GRACIOUS GRAVY! The carnage is absolutely darling!",
	"HOLY GUACAMOLE! Their skeletal system just got reorganized!",
	"GOSH DARN TOOTIN'! That's premium-quality mayhem right there!",
	"WELL I'LL BE HORNSWOGGLED! What magnificent destruction!",
	"GREAT GALLOPING GALOSHES! The violence is simply divine!",
	"SWEET SUFFERING SUCCOTASH! Someone's organs got shuffled!",
	"GOLLY MOLLY! That's what I call therapeutic violence!",
	"HOLY TOLEDO! Their molecular bonds just said goodbye!",
	"SWEET SASSY MOLASSY! The brutality is absolutely charming!",
	"GOOD LORD ALMIGHTY! That's some Grade-A carnage!",
	"HOLY COW PATTIES! Someone's getting their chakras realigned!",
	"GOSH GOLLY GEE WILLIKERS! The destruction is so heartwarming!",
	"SWEET BUTTERY BISCUITS! That's championship-caliber violence!",
	"GREAT BALLS OF FIRE! Their consciousness just took a detour!",
	"HOLY MOLY RAVIOLI! What delightfully barbaric entertainment!",
	"JUMPING JELLY BEANS! Someone's getting their aura adjusted!",
	"SWEET GEORGIA PEACHES! The carnage is absolutely precious!",
	"GOLLY GEE WHILLIKERS! That's some family-friendly brutality!",
	"HOLY GUACAMOLE BATMAN! Their life force just got renovated!",
	"GREAT LEAPING LIZARDS! What magnificently violent artistry!",
	"SWEET MERCIFUL MOSES! The destruction is so wholesome!",
	"GOOD GOLLY GOSH DARN! Someone's getting their essence purified!",
	"HOLY JUMPING JACK FLASH! That's therapeutic-grade violence!",
	"JIMINY CRICKET CRACKERS! The brutality is absolutely adorable!",
	"SWEET SAINTED SAINTS! Their molecular structure got a makeover!",
	"HOLY MOLY MACARONI! Someone's nervous system just got updated!",
	"GREAT GALLOPING GUMMY BEARS! The violence is so refreshing!",
	"SWEET BABY JESUS ON A POGO STICK! That's premium mayhem!",
	"GOOD GRAVY TRAIN! Their atoms just got a spring cleaning!",
	"JUMPING JALAPEÑOS! Someone's getting their chi realigned!",
	"SWEET SUFFERING SIDEWINDERS! The carnage is absolutely lovely!",
	"GOLLY GEE BUTTERSCOTCH! That's some therapeutic bone-breaking!",
	"HOLY MOLY PEPPERONI! Their consciousness just got defragmented!",
	"GOOD GOOGLY MOOGLY! What wonderfully violent therapy!",
	"SWEET SASSY FRASSY! The destruction is so darn cute!",
	"GOOD GOLLY MOLLY WOLLY! Someone's getting their chakras dusted!",
	"HOLY JUMPING JACKFRUIT! That's Grade-A premium violence!",
	"JIMINY CHRISTMAS COOKIES! The brutality is absolutely darling!",
	"SWEET MERCIFUL MACAROONS! Their life force just got refreshed!",
	"GOLLY WILLIKERS WHISKERS! What magnificently violent wellness!",
	"HOLY MOLY GUACAMOLE CANNOLI! The carnage is so therapeutic!",
	"GREAT GALLOPING GALOSHES AND GARTERS! That's beautiful brutality!",
	"SWEET SAINTED SUCCOTASH SANDWICHES! Someone's getting soul maintenance!",
	"GOOD GRAVY BOATS AND BISCUITS! The violence is absolutely precious!",
	"HOLY SHIITAKE SHAKE AND BAKE! Their essence just got steam-cleaned!",
	"JUMPING JELLY BEAN JALAPEÑOS! What delightfully violent self-care!",
	"SWEET BABY BUTTERSCOTCH BANANAS! The destruction is so wholesome!",
}

var doctorComments = []string{
	"From a scientific perspective, that spleen is COMPLETELY destroyed!",
	"The molecular density of that impact was off the charts!",
	"Fascinating! Their blood type is clearly superior in this exchange!",
	"That level of existential dread should be medically impossible!",
	"The horoscope alignment is causing unprecedented violence!",
	"I'm detecting severe trauma to the chaos dimension!",
	"That's what happens when you ignore the laws of physics!",
	"Remarkable! The cranial displacement exceeds all theoretical models!",
	"According to my calculations, that should have been fatal!",
	"The biomechanical stress patterns are absolutely fascinating!",
	"I'm observing complete cellular restructuring in real-time!",
	"The neurological impact registers at 47.3 chaos units!",
	"Extraordinary! Their pain receptors have transcended mortal limitations!",
	"My instruments are detecting quantum-level bone fragmentation!",
	"The psychological trauma coefficient is approaching infinity!",
	"Clinically speaking, that was a textbook reality fracture!",
	"The metabolic disruption patterns are beautifully symmetrical!",
	"I'm witnessing unprecedented damage to their space-time continuum!",
	"The kinetic energy transfer violated three fundamental constants!",
	"Medically speaking, they just got scientifically obliterated!",
	"The anatomical impossibility quotient is exceeding safe parameters!",
	"Fascinating! The mitochondrial degradation is proceeding as predicted!",
	"I'm recording catastrophic failure in their skeletal matrix!",
	"The endocrine system appears to be completely discombobulated!",
	"Remarkable! Their DNA helixes are unwinding in perfect spirals!",
	"The cardiovascular disruption is creating beautiful fluid dynamics!",
	"I'm observing complete synaptic meltdown across all neural pathways!",
	"The respiratory system has achieved maximum entropy coefficient!",
	"Extraordinary! Their lymphatic network is restructuring itself!",
	"The muscular tissue is exhibiting impossible contraction patterns!",
	"Clinically fascinating! Complete organ system cascade failure!",
	"The cerebrospinal fluid pressure has exceeded all known limits!",
	"I'm detecting massive hemorrhaging in seventeen different locations!",
	"The bone marrow composition is undergoing rapid metamorphosis!",
	"Remarkable! Their nervous system is rewiring itself in real-time!",
	"The cellular regeneration rate has dropped to negative integers!",
	"I'm witnessing complete molecular dissociation at the atomic level!",
	"The blood-brain barrier has suffered catastrophic structural failure!",
	"Extraordinary! Their consciousness appears to be leaking!",
	"The digestive tract is exhibiting reverse peristalsis patterns!",
	"I'm recording unprecedented damage to their proprioceptive sensors!",
	"The immune system has gone into complete defensive shutdown!",
	"Fascinating! Their temporal lobe is processing memories backwards!",
	"The vertebral column shows signs of interdimensional compression!",
	"I'm observing complete cellular mitosis failure across all tissues!",
	"The hypothalamic-pituitary axis has suffered total collapse!",
	"Remarkable! Their adenosine triphosphate production has ceased!",
	"The corneal reflex indicates severe brainstem trauma!",
	"I'm detecting massive protein denaturation in muscle fibers!",
	"The autonomic nervous system is exhibiting chaotic oscillations!",
	"Extraordinary! Their reticular formation is completely scrambled!",
	"The hepatic enzymes are catalyzing in reverse chemical reactions!",
	"I'm witnessing complete dermal integrity failure!",
	"The pulmonary alveoli show signs of spontaneous implosion!",
	"Fascinating! Their cochlear nerve is transmitting impossible frequencies!",
	"The renal filtration system has achieved negative efficiency!",
	"I'm recording complete breakdown of their blood coagulation cascade!",
	"The pancreatic islets are secreting anti-insulin compounds!",
	"Remarkable! Their motor cortex is firing random chaos patterns!",
	"The thyroid gland appears to be producing temporal hormones!",
	"I'm observing massive disruption to their calcium-sodium pumps!",
	"The olfactory bulb is processing smells from parallel dimensions!",
	"Extraordinary! Their pineal gland is secreting liquid darkness!",
	"The adrenal cortex is producing impossible stress hormones!",
	"I'm detecting complete failure of their hemoglobin oxygen transport!",
	"The cerebellum shows signs of gravitational anomalies!",
	"Fascinating! Their appendix has suddenly become medically relevant!",
	"The spinal cord is conducting electrical impulses backwards!",
	"I'm witnessing complete breakdown of their cellular membrane integrity!",
	"The pituitary gland is secreting growth hormone at quantum levels!",
	"Remarkable! Their gallbladder is producing bile in three dimensions!",
	"The medulla oblongata shows signs of temporal displacement!",
	"I'm observing complete failure of their sodium-potassium gradients!",
	"The parathyroid glands are regulating impossible calcium levels!",
	"Extraordinary! Their bone density has achieved negative mass!",
	"The corpus callosum is transferring thoughts to alternate realities!",
	"I'm detecting massive disruption to their circadian rhythm proteins!",
	"The thymus gland appears to be aging in reverse!",
	"Fascinating! Their stem cells are differentiating into chaos particles!",
	"The vagus nerve is transmitting signals to their past self!",
	"I'm witnessing complete breakdown of their electron transport chain!",
	"The hippocampus is storing memories that haven't happened yet!",
	"Remarkable! Their lymph nodes are filtering interdimensional toxins!",
	"The brainstem is regulating functions that don't exist!",
	"I'm observing complete cellular apoptosis across all organ systems!",
	"The endoplasmic reticulum has achieved impossible protein folding!",
	"Extraordinary! Their Golgi apparatus is packaging pure violence!",
	"The ribosomes are translating RNA into existential poetry!",
	"I'm detecting massive failure in their ATP synthase complexes!",
	"The peroxisomes are oxidizing hope itself!",
	"Fascinating! Their lysosomes are digesting their own reality!",
	"The cytoskeleton has collapsed into a quantum probability cloud!",
	"I'm witnessing complete nuclear membrane dissolution!",
	"The telomeres are unraveling faster than space-time itself!",
}

var sallyComments = []string{
	"I'VE NEVER SEEN SUCH BEAUTIFUL CARNAGE!",
	"YES! MORE VIOLENCE! FEED THE CHAOS GODS!",
	"THAT'S HOW YOU EMBRACE THE EXISTENTIAL VOID!",
	"BLOOD FOR THE BLOOD DIMENSION!",
	"THIS IS PEAK HUMAN PERFORMANCE!",
	"I'M LITERALLY CRYING TEARS OF JOY!",
	"DESTROY THEM! OBLITERATE THEIR MOLECULAR STRUCTURE!",
	"MAGNIFICENT BRUTALITY! THE CHAOS SPIRITS ARE PLEASED!",
	"YESSSSS! CRUSH THEIR VERY ESSENCE INTO STARDUST!",
	"BEAUTIFUL! ABSOLUTELY BEAUTIFUL DESTRUCTION!",
	"TEAR APART THE FABRIC OF REALITY ITSELF!",
	"THIS IS WHAT TRUE ARTISTRY LOOKS LIKE!",
	"SHATTER THEIR BONES INTO A THOUSAND PIECES!",
	"THE VIOLENCE IS SO PURE! SO TRANSCENDENT!",
	"ANNIHILATE THEIR HOPES AND DREAMS!",
	"GRIND THEIR SPIRIT INTO COSMIC POWDER!",
	"YES! MAKE THEM REGRET EXISTING!",
	"THE PAIN! THE GLORIOUS, GLORIOUS PAIN!",
	"DEVASTATE THEIR MOLECULAR COMPOSITION!",
	"THIS IS BETTER THAN CHRISTMAS MORNING!",
	"PULVERIZE THEIR VERY CONCEPT OF SELF!",
	"THE CHAOS DIMENSION HUNGERS FOR MORE!",
	"MAGNIFICENT! ABSOLUTELY MAGNIFICENT CARNAGE!",
	"TURN THEIR SKELETON INTO ABSTRACT ART!",
	"YES! ERASE THEM FROM THE TIMELINE!",
	"BEAUTIFUL SUFFERING! EXQUISITE AGONY!",
	"DEMOLISH THEIR FAITH IN PHYSICS!",
	"THIS IS POETRY WRITTEN IN VIOLENCE!",
	"SCATTER THEIR ATOMS ACROSS THE UNIVERSE!",
	"THE BRUTALITY IS SO AESTHETICALLY PLEASING!",
	"CRUSH THEIR DREAMS INTO FINE POWDER!",
	"YES! MAKE REALITY ITSELF WEEP!",
	"DISINTEGRATE THEIR SENSE OF PURPOSE!",
	"THE CARNAGE IS ABSOLUTELY SUBLIME!",
	"TURN THEIR NERVOUS SYSTEM INTO CONFETTI!",
	"BEAUTIFUL! REDUCE THEM TO QUANTUM FOAM!",
	"YES! VIOLATE THE LAWS OF NATURE!",
	"SHRED THEIR CONSCIOUSNESS INTO RIBBONS!",
	"THIS IS MAXIMUM THERAPEUTIC VIOLENCE!",
	"OBLITERATE THEIR WILL TO LIVE!",
	"THE DESTRUCTION IS SO ROMANTICALLY VIOLENT!",
	"YES! MAKE THE VOID ITSELF JEALOUS!",
	"PULVERIZE THEIR CHILDHOOD MEMORIES!",
	"BEAUTIFUL CHAOS! MAGNIFICENT MAYHEM!",
	"TURN THEIR HOPE INTO LIQUID DESPAIR!",
	"YES! CRUSH THEIR SPIRIT LIKE A GRAPE!",
	"DEMOLISH THEIR FAITH IN EXISTENCE!",
	"THE VIOLENCE IS SO TRANSCENDENTALLY PURE!",
	"SCATTER THEIR ESSENCE TO THE WINDS!",
	"YES! MAKE THEIR ANCESTORS FEEL SHAME!",
	"BEAUTIFUL! TURN THEIR BONES TO DUST!",
	"OBLITERATE THEIR SENSE OF REALITY!",
	"THE CARNAGE IS ABSOLUTELY ORGASMIC!",
	"YES! FEED THEIR PAIN TO THE DARKNESS!",
	"MAGNIFICENT! REDUCE THEM TO PARTICLES!",
	"BEAUTIFUL SUFFERING! DIVINE DESTRUCTION!",
	"YES! MAKE THE UNIVERSE ITSELF SCREAM!",
	"PULVERIZE THEIR HOPES AND ASPIRATIONS!",
	"THE VIOLENCE IS SO ARTISTICALLY PERFECT!",
	"YES! TURN THEIR SOUL INTO VAPOR!",
	"BEAUTIFUL! CRUSH THEIR VERY ESSENCE!",
	"OBLITERATE THEIR CONNECTION TO REALITY!",
	"THE DESTRUCTION IS ABSOLUTELY INTOXICATING!",
	"YES! MAKE THEM QUESTION EXISTENCE ITSELF!",
	"MAGNIFICENT! TEAR APART THEIR TIMELINE!",
	"BEAUTIFUL CHAOS! PERFECT PANDEMONIUM!",
	"YES! REDUCE THEM TO PRIMORDIAL SOUP!",
	"DEMOLISH THEIR FAITH IN MATHEMATICS!",
	"THE CARNAGE IS SO SPIRITUALLY FULFILLING!",
	"YES! TURN THEIR DREAMS INTO NIGHTMARES!",
	"BEAUTIFUL! OBLITERATE THEIR SENSE OF SELF!",
	"MAGNIFICENT! CRUSH THEIR ATOMIC STRUCTURE!",
	"YES! MAKE REALITY ITSELF APOLOGIZE!",
	"PULVERIZE THEIR BELIEF IN TOMORROW!",
	"THE VIOLENCE IS SO EMOTIONALLY SATISFYING!",
	"YES! TURN THEIR CONSCIOUSNESS TO MIST!",
	"BEAUTIFUL DESTRUCTION! PERFECT PAIN!",
	"OBLITERATE THEIR FAITH IN GRAVITY!",
	"THE BRUTALITY IS SO AESTHETICALLY DIVINE!",
	"YES! MAKE THE COSMOS ITSELF WEEP!",
	"MAGNIFICENT! REDUCE THEM TO ENERGY!",
	"BEAUTIFUL! CRUSH THEIR DIMENSIONAL STABILITY!",
	"YES! TURN THEIR MEMORIES INTO STATIC!",
	"DEMOLISH THEIR TRUST IN CAUSALITY!",
	"THE CARNAGE IS SO PHILOSOPHICALLY PURE!",
	"YES! OBLITERATE THEIR QUANTUM COHERENCE!",
	"BEAUTIFUL SUFFERING! MAGNIFICENT MISERY!",
	"TURN THEIR SANITY INTO ABSTRACT CONCEPTS!",
	"YES! MAKE ENTROPY ITSELF JEALOUS!",
	"PULVERIZE THEIR FAITH IN LINEAR TIME!",
	"THE VIOLENCE IS SO METAPHYSICALLY CORRECT!",
	"YES! REDUCE THEM TO PURE MATHEMATICS!",
	"BEAUTIFUL! CRUSH THEIR EXISTENTIAL FRAMEWORK!",
	"MAGNIFICENT! OBLITERATE THEIR SPACETIME!",
	"YES! TURN THEIR REALITY INTO POETRY!",
}

var commissionerComments = []string{
	"The Department approves of this violence level.",
	"This combat meets our chaos quotas.",
	"Violence parameters are within acceptable ranges.",
	"The Commissioner is... pleased.",
	"Existential dread levels: OPTIMAL.",
	"This fighter shows proper Department training.",
	"Authorization granted for maximum carnage.",
	"Efficiency rating: EXEMPLARY.",
	"The Department commends this display of brutality.",
	"Violence metrics exceed minimum requirements.",
	"Combat effectiveness: SATISFACTORY.",
	"The Commissioner notes this for future reference.",
	"Department protocol 7-Alpha has been satisfied.",
	"This level of destruction is... adequate.",
	"Violence quotient approved by upper management.",
	"The Department's expectations have been met.",
	"Combat proficiency: ACCEPTABLE.",
	"The Commissioner expresses mild satisfaction.",
	"This violence has been properly catalogued.",
	"Department regulations require this level of carnage.",
	"The Commissioner's approval rating increases marginally.",
	"Violence standards maintained within Department guidelines.",
	"This combat efficiency pleases the bureaucratic overlords.",
	"The Department's violence metrics have been updated.",
	"Combat authorization code: CONFIRMED.",
	"The Commissioner observes. The Commissioner remembers.",
	"This violence aligns with projected outcomes.",
	"Department protocol 12-Gamma is now in effect.",
	"The Commissioner's interest is... piqued.",
	"Violence levels calibrated to optimal parameters.",
	"This combat serves the Department's interests.",
	"The Commissioner makes note of this efficiency.",
	"Department standard 47-Delta has been exceeded.",
	"This violence is consistent with our projections.",
	"The Commissioner's database has been updated.",
	"Combat effectiveness falls within expected margins.",
	"The Department requires documentation of this event.",
	"Violence authorization: PERMANENTLY APPROVED.",
	"The Commissioner finds this... instructive.",
	"Department oversight confirms satisfactory brutality.",
	"This combat adheres to regulation 23-Echo.",
	"The Commissioner's algorithms approve this outcome.",
	"Violence metrics synchronized with central database.",
	"The Department acknowledges this display of force.",
	"Combat efficiency rating: ABOVE STANDARD.",
	"The Commissioner's surveillance confirms compliance.",
	"Department protocol demands this level of violence.",
	"This brutality satisfies administrative requirements.",
	"The Commissioner's analysis indicates optimal performance.",
	"Violence parameters locked in at current settings.",
	"The Department's quarterly goals are being met.",
	"Combat authorization renewed indefinitely.",
	"The Commissioner observes without judgment.",
	"Department regulation 88-Foxtrot is now active.",
	"This violence serves purposes beyond your understanding.",
	"The Commissioner's attention is... focused.",
	"Department standards require this caliber of destruction.",
	"Combat effectiveness synchronized with master timeline.",
	"The Commissioner makes adjustments to future projections.",
	"Violence levels optimal for current experimental phase.",
	"The Department's long-term objectives advance accordingly.",
	"Combat authorization escalated to Level Seven.",
	"The Commissioner's contingency plans remain unchanged.",
	"Department oversight confirms regulatory compliance.",
	"This violence aligns with predetermined trajectories.",
	"The Commissioner's approval is... noted.",
	"Combat effectiveness exceeds baseline requirements.",
	"The Department acknowledges superior execution.",
	"Violence parameters adjusted for future iterations.",
	"The Commissioner's calculations prove accurate.",
	"Department protocol 99-Hotel is hereby activated.",
	"This brutality serves the greater administrative framework.",
	"Combat authorization granted across all timelines.",
	"The Commissioner observes. The Commissioner learns.",
	"Department standards maintained at acceptable levels.",
	"Violence metrics uploaded to central processing.",
	"The Commissioner's interest remains... professional.",
	"Combat effectiveness validates current methodologies.",
	"The Department's experimental parameters are satisfied.",
	"Violence authorization permanent and irrevocable.",
	"The Commissioner notes correlation with previous data.",
	"Department oversight confirms projected outcomes.",
	"This combat serves purposes you cannot comprehend.",
	"The Commissioner's database expands accordingly.",
	"Violence levels consistent with administrative needs.",
	"Combat authorization transcends temporal boundaries.",
	"The Department acknowledges this statistical anomaly.",
	"The Commissioner's silence speaks volumes.",
	"Department protocol demands escalation to Phase Two.",
	"Violence parameters exceed all safety regulations.",
	"The Commissioner observes patterns within the chaos.",
	"Combat effectiveness validates the Department's methods.",
	"The Department's influence grows with each impact.",
	"Violence authorization granted retroactively and prophetically.",
	"The Commissioner's approval echoes across dimensions.",
	"Department oversight confirms reality manipulation success.",
	"This brutality advances the Commissioner's timeline.",
}

// GenerateLiveAction creates a dramatic description of a fight tick
func GenerateLiveAction(fightID, tickNumber int, fighter1, fighter2 database.Fighter, damage1, damage2, health1, health2, round int) LiveAction {
	seed := utils.FightTickSeed(fightID, tickNumber)
	rng := rand.New(rand.NewSource(seed))

	var action LiveAction
	action.Type = "damage"
	action.TickNumber = tickNumber
	action.Round = round
	action.Health1 = health1
	action.Health2 = health2

	// Randomly pick which fighter to announce (since both are dealing damage)
	if rng.Intn(2) == 0 {
		// Announce Fighter1's attack on Fighter2
		action.Attacker = fighter1.Name
		action.Victim = fighter2.Name
		action.Damage = damage2
	} else {
		// Announce Fighter2's attack on Fighter1
		action.Attacker = fighter2.Name
		action.Victim = fighter1.Name
		action.Damage = damage1
	}

	// Generate dramatic action description
	combatAction := combatActions[rng.Intn(len(combatActions))]
	action.Action = fmt.Sprintf("%s! %s connects for %s damage!",
		combatAction, action.Attacker, formatNumber(action.Damage))

	// Check for special events
	if action.Damage > 4000 {
		action.Type = "critical"
		action.Action = "🔥 CRITICAL HIT! " + action.Action
	}

	if health1 <= 20000 || health2 <= 20000 {
		action.Type = "low_health"
	}

	// Add announcer commentary
	announcer := announcers[rng.Intn(len(announcers))]
	action.Announcer = announcer.Name

	switch announcer.Name {
	case "Chad Puncherson":
		action.Commentary = chadComments[rng.Intn(len(chadComments))]
	case "Dr. Mayhem PhD":
		action.Commentary = doctorComments[rng.Intn(len(doctorComments))]
	case "\"Screaming\" Sally Bloodworth":
		action.Commentary = sallyComments[rng.Intn(len(sallyComments))]
	case "The Commissioner":
		action.Commentary = commissionerComments[rng.Intn(len(commissionerComments))]
	}

	return action
}

// GenerateDeathAction creates a special death announcement
func GenerateDeathAction(fightID int, winner, loser database.Fighter, health1, health2, round int) LiveAction {
	seed := utils.FightTickSeed(fightID, 999999) // Special seed for death
	rng := rand.New(rand.NewSource(seed))

	deathMessages := []string{
		"FATALITY! %s's existential dread has reached MAXIMUM CAPACITY!",
		"GAME OVER! %s has been sent to the CHAOS DIMENSION!",
		"OBLITERATION! %s's molecular structure has COLLAPSED!",
		"ANNIHILATION! %s has achieved the ultimate existential crisis!",
		"DESTRUCTION! %s's blood type couldn't save them now!",
	}

	message := deathMessages[rng.Intn(len(deathMessages))]

	return LiveAction{
		Type:       "death",
		Action:     fmt.Sprintf(message, loser.Name),
		Attacker:   winner.Name,
		Victim:     loser.Name,
		Health1:    health1,
		Health2:    health2,
		Round:      round,
		Announcer:  "\"Screaming\" Sally Bloodworth",
		Commentary: "THIS IS THE MOST BEAUTIFUL VIOLENCE I'VE EVER WITNESSED!",
	}
}

// GenerateRoundAction creates round transition announcements
func GenerateRoundAction(round int, health1, health2 int) LiveAction {
	return LiveAction{
		Type:       "round",
		Action:     fmt.Sprintf("🔥 ROUND %d BEGINS! The violence escalates to unprecedented levels! 🔥", round),
		Health1:    health1,
		Health2:    health2,
		Round:      round,
		Announcer:  "Chad Puncherson",
		Commentary: "Here we go again! More beautiful chaos incoming!",
	}
}

func formatNumber(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%s", addCommas(n))
	}
	return fmt.Sprintf("%d", n)
}

func addCommas(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}
