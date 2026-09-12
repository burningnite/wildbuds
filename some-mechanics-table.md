# Elements

| 𖢥 | beast |
| - | ----- |
| 𖧷 | flora |
| 𖦹 | water |
| 𖥳 | fire  |
| 𖣯 | earth |
| 𖣘 | air   |
| 𖥑 | cold  |
| 𖢒 | metal |
| 𖡷 | void  |
| 𖤓 | gleam |

# Interactions

Where top row is defender, and left column is attacker ( first data point is a Beast Type attack against a Beast type creature, second is a Beast type attack against a Flora type creature, and so on)

| TYPES | beast | flora | water | fire | earth | air | cold | metal | void | gleam |
| ----- | ----- | ----- | ----- | ---- | ----- | --- | ---- | ----- | ---- | ----- |
| beast | ↑     | ↑     | ↑     | ≡    | ≡     | ↓   | ≡    | ↓     | ↓    | ≡     |
| flora | ↓     | ≡     | ↑     | ↓    | ↑     | ≡   | ↓    | ≡     | ≡    | ↑     |
| water | ≡     | ↓     | ↓     | ↑    | ≡     | ↓   | ↑    | ≡     | ↑    | ≡     |
| fire  | ≡     | ↑     | ↓     | ↓    | ≡     | ≡   | ↑    | ↑     | ≡    | ↓     |
| earth | ≡     | ↓     | ≡     | ↑    | ↓     | ≡   | ↓    | ≡     | ↑    | ↑     |
| air   | ↑     | ≡     | ↑     | ↓    | ≡     | ↑   | ≡    | ↓     | ↓    | ≡     |
| cold  | ↓     | ≡     | ↓     | ≡    | ↑     | ↑   | ↓    | ↑     | ≡    | ≡     |
| metal | ≡     | ≡     | ≡     | ≡    | ↑     | ≡   | ↑    | ↓     | ↓    | ≡     |
| void  | ↑     | ↑     | ≡     | ↑    | ↓     | ↓   | ≡    | ≡     | ≡    | ↓     |
| gleam | ↓     | ↓     | ≡     | ≡    | ↓     | ↑   | ≡    | ↑     | ↑    | ≡     |

Where ↑ is SUPER , ↓ is LESS, and  ≡ is NORMAL. (super-effective, less-effective, normal). Mechanics of this will be dealt later. Dual type stack these weaknesses and/or resistances; here's a table with some examples (just a few entries, not taxative). Here the header is the attack type, and the first two columns are the Defending creature's typing (if type1 is equal to type2 it just means it's a monotype Beast creature and therefore doesn't stack it's weaknesses).

| type1 | type2 | beast | flora | earth | fire | water | air | cold | metal | void | gleam |
| ----- | ----- | ----- | ----- | ----- | ---- | ----- | --- | ---- | ----- | ---- | ----- |
| beast | beast | ↑     | ↓     | ≡     | ≡    | ≡     | ↑   | ↓    | ≡     | ↑    | ↓     |
| beast | flora | ↑↑    | ↓     | ↓     | ↑    | ↓     | ↑   | ↓    | ≡     | ↑↑   | ↓↓    |
| beast | earth | ↑     | ≡     | ↓     | ≡    | ≡     | ↑   | ≡    | ↑     | ≡    | ↓↓    |
| beast | fire  | ↑     | ↓↓    | ↑     | ↓    | ↑     | ≡   | ↓    | ≡     | ↑↑   | ↓     |
| beast | water | ↑↑    | ≡     | ≡     | ↓    | ↓     | ↑↑  | ↓↓   | ≡     | ↑    | ↓     |
| beast | air   | ≡     | ↓     | ≡     | ≡    | ↓     | ↑↑  | ≡    | ≡     | ≡    | ≡     |
| beast | cold  | ↑     | ↓↓    | ↓     | ↑    | ↑     | ↑   | ↓↓   | ↑     | ↑    | ↓     |
| beast | metal | ≡     | ↓     | ≡     | ↑    | ≡     | ≡   | ≡    | ↓     | ↑    | ≡     |
| beast | void  | ≡     | ↓     | ↑     | ≡    | ↑     | ≡   | ↓    | ↓     | ↑    | ≡     |
| beast | gleam | ↑     | ≡     | ↑     | ↓    | ≡     | ↑   | ↓    | ≡     | ≡    | ↓     |

Here we have additional effectiveness tags: ↑↑ is MEGA , ↓↓ is LEAST . An additional tag is achievable using the same principle as Pokemon's STAB, if a Creature uses an ACTION ( unlike pokemon, STAB here wouldn't just apply to attacks) of the same typing as one of it's own ( Beast Metal Creature uses a Metal ACTION or Beast ACTION), then that ACTION is considered one ↑ higher : ↓↓ becomes ↓, ↓ becomes ≡, ≡ becomes ↑, ↑ becomes ↑↑ AND ↑↑ becomes ↑↑↑. ↑↑↑ is ULTRA (↑↑ mega-effective, ↓↓ least-effective, ↑↑↑ ultra-effective)

| symbol | name   | +stab |
| ------ | ------ | ----- |
| ↓↓     | LEAST  | ↓     |
| ↓      | LESS   | ≡     |
| ≡      | NORMAL | ↑     |
| ↑      | SUPER  | ↑↑    |
| ↑↑     | MEGA   | ↑↑↑   |
| ↑↑↑    | ULTRA  | ↑↑↑   |

