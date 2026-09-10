#![allow(dead_code)]
use bevy::prelude::*;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Element {
    Normal,
    Fire,
    Water,
    Grass,
    Electric,
    Ice,
    Fighting,
    Poison,
    Ground,
    Flying,
    Psychic,
    Bug,
    Rock,
    Ghost,
    Dragon,
    Dark,
    Steel,
    Fairy,
}

#[derive(Component, Debug, Clone)]
pub struct DualType(pub Element, pub Option<Element>);

pub fn get_multiplier(attack_type: Element, target_type: Element) -> f32 {
    if attack_type == target_type {
        return 0.5; // Basic resistance to own type
    }
    match (attack_type, target_type) {
        (Element::Water, Element::Fire) => 2.0,
        (Element::Fire, Element::Grass) => 2.0,
        (Element::Grass, Element::Water) => 2.0,
        // TODO: Expand the matrix
        _ => 1.0,
    }
}

pub fn calculate_multiplier(attack_type: Element, target_types: &DualType) -> f32 {
    let mut mult = get_multiplier(attack_type, target_types.0);
    if let Some(second_type) = target_types.1 {
        mult *= get_multiplier(attack_type, second_type);
    }
    mult
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dual_type_multiplier() {
        let target = DualType(Element::Fire, Some(Element::Grass)); 
        assert_eq!(calculate_multiplier(Element::Water, &target), 2.0 * 1.0); 

        let target2 = DualType(Element::Fire, None);
        assert_eq!(calculate_multiplier(Element::Water, &target2), 2.0);
    }
}
