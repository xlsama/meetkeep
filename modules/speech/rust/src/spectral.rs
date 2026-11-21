use faer::{Mat, Side};
use std::collections::{BTreeMap, BTreeSet};

const PVAL: f64 = 0.012;
const MIN_PRUNE_NEIGHBORS: usize = 6;
const MAX_SPEAKERS: usize = 15;
const CLUSTER_LINE: usize = 10;
const MAX_CLUSTER_SAMPLES: usize = 6000;
const MIN_CLUSTER_SIZE: usize = 4;
const MERGE_COSINE_THRESHOLD: f32 = 0.875;

pub fn labels(embeddings: &[Vec<f32>], speakers: Option<usize>) -> Vec<usize> {
    labels_capped(embeddings, speakers, MAX_CLUSTER_SAMPLES)
}

fn labels_capped(embeddings: &[Vec<f32>], speakers: Option<usize>, cap: usize) -> Vec<usize> {
    let n = embeddings.len();
    if n == 0 {
        return Vec::new();
    }
    if n == 1 || speakers.is_some_and(|k| k <= 1) || (speakers.is_none() && n < CLUSTER_LINE) {
        return vec![0; n];
    }
    if n > cap {
        return subsampled(embeddings, speakers, cap);
    }

    let mut matrix = cosine_similarity(embeddings);
    prune(&mut matrix, n);
    laplacian(&mut matrix, n);
    let dense = Mat::from_fn(n, n, |i, j| matrix[i * n + j]);
    drop(matrix);
    let Ok(eigen) = dense.self_adjoint_eigen(Side::Lower) else {
        return vec![0; n];
    };
    let values: Vec<f32> = eigen.S().column_vector().iter().copied().collect();
    let vectors = eigen.U();

    let k = match speakers {
        Some(k) => k.min(n),
        None => estimate_speakers(&values),
    };
    if k <= 1 {
        return vec![0; n];
    }
    let mut points = vec![0f32; n * k];
    for i in 0..n {
        for j in 0..k {
            points[i * k + j] = vectors[(i, j)];
        }
    }
    let raw = kmeans(&points, n, k, k);
    if speakers.is_some() {
        raw
    } else {
        merge_by_cosine(filter_minor(raw, embeddings), embeddings)
    }
}

fn subsampled(embeddings: &[Vec<f32>], speakers: Option<usize>, cap: usize) -> Vec<usize> {
    let n = embeddings.len();
    let stride = n.div_ceil(cap);
    let sampled: Vec<Vec<f32>> = embeddings.iter().step_by(stride).cloned().collect();
    let sampled_labels = labels_capped(&sampled, speakers, cap);
    let centers: BTreeMap<usize, Vec<f32>> = members(&sampled_labels)
        .into_iter()
        .map(|(label, indices)| (label, mean_center(&sampled, &indices)))
        .collect();
    if centers.len() <= 1 {
        return vec![0; n];
    }
    embeddings.iter().map(|v| nearest(v, &centers, 0)).collect()
}

fn nearest(vector: &[f32], centers: &BTreeMap<usize, Vec<f32>>, fallback: usize) -> usize {
    let mut best = (fallback, -2f32);
    for (label, center) in centers {
        let sim = cosine(vector, center);
        if sim > best.1 {
            best = (*label, sim);
        }
    }
    best.0
}

fn members(labels: &[usize]) -> BTreeMap<usize, Vec<usize>> {
    let mut out: BTreeMap<usize, Vec<usize>> = BTreeMap::new();
    for (i, l) in labels.iter().enumerate() {
        out.entry(*l).or_default().push(i);
    }
    out
}

fn mean_center(embeddings: &[Vec<f32>], indices: &[usize]) -> Vec<f32> {
    let mut center = vec![0f32; embeddings[0].len()];
    for &i in indices {
        for (c, v) in center.iter_mut().zip(&embeddings[i]) {
            *c += v;
        }
    }
    let scale = 1.0 / indices.len() as f32;
    center.iter_mut().for_each(|c| *c *= scale);
    center
}

fn cosine(a: &[f32], b: &[f32]) -> f32 {
    let (mut dot, mut na, mut nb) = (0f32, 0f32, 0f32);
    for (x, y) in a.iter().zip(b) {
        dot += x * y;
        na += x * x;
        nb += y * y;
    }
    dot / (na.sqrt() * nb.sqrt()).max(1e-9)
}

fn filter_minor(labels: Vec<usize>, embeddings: &[Vec<f32>]) -> Vec<usize> {
    let by_cluster = members(&labels);
    let minor: BTreeSet<usize> = by_cluster
        .iter()
        .filter(|(_, v)| v.len() < MIN_CLUSTER_SIZE)
        .map(|(k, _)| *k)
        .collect();
    if minor.is_empty() {
        return labels;
    }
    let centers: BTreeMap<usize, Vec<f32>> = by_cluster
        .iter()
        .filter(|(k, _)| !minor.contains(k))
        .map(|(k, v)| (*k, mean_center(embeddings, v)))
        .collect();
    if centers.is_empty() {
        return vec![0; labels.len()];
    }
    labels
        .iter()
        .enumerate()
        .map(|(i, l)| if minor.contains(l) { nearest(&embeddings[i], &centers, *l) } else { *l })
        .collect()
}

fn merge_by_cosine(mut labels: Vec<usize>, embeddings: &[Vec<f32>]) -> Vec<usize> {
    loop {
        let by_cluster = members(&labels);
        if by_cluster.len() <= 1 {
            break;
        }
        let keys: Vec<usize> = by_cluster.keys().copied().collect();
        let centers: Vec<Vec<f32>> = keys.iter().map(|k| mean_center(embeddings, &by_cluster[k])).collect();
        let mut best = (0, 0, -2f32);
        for i in 0..keys.len() {
            for j in i + 1..keys.len() {
                let sim = cosine(&centers[i], &centers[j]);
                if sim > best.2 {
                    best = (i, j, sim);
                }
            }
        }
        if best.2 < MERGE_COSINE_THRESHOLD {
            break;
        }
        let (into, from) = (keys[best.0], keys[best.1]);
        labels.iter_mut().filter(|l| **l == from).for_each(|l| *l = into);
    }
    labels
}

fn estimate_speakers(values: &[f32]) -> usize {
    let gaps = MAX_SPEAKERS.min(values.len().saturating_sub(1));
    let mut best = (0, f32::NEG_INFINITY);
    for i in 0..gaps {
        let gap = values[i + 1] - values[i];
        if gap > best.1 {
            best = (i, gap);
        }
    }
    best.0 + 1
}

fn cosine_similarity(embeddings: &[Vec<f32>]) -> Vec<f32> {
    let n = embeddings.len();
    let d = embeddings[0].len();
    let rows = Mat::from_fn(n, d, |i, j| {
        let e = &embeddings[i];
        let norm = e.iter().map(|v| v * v).sum::<f32>().sqrt().max(1e-9);
        e[j] / norm
    });
    let sim = &rows * rows.transpose();
    let mut out = vec![0f32; n * n];
    for i in 0..n {
        for j in 0..n {
            out[i * n + j] = sim[(i, j)];
        }
    }
    out
}

fn prune(matrix: &mut [f32], n: usize) {
    let zeroed = (((1.0 - PVAL) * n as f64) as usize).min(n.saturating_sub(MIN_PRUNE_NEIGHBORS));
    if zeroed == 0 {
        return;
    }
    let mut order: Vec<usize> = Vec::with_capacity(n);
    for row in matrix.chunks_mut(n) {
        order.clear();
        order.extend(0..n);
        order.sort_by(|a, b| row[*a].total_cmp(&row[*b]));
        for &i in &order[..zeroed] {
            row[i] = 0.0;
        }
    }
}

fn laplacian(matrix: &mut [f32], n: usize) {
    for i in 0..n {
        matrix[i * n + i] = 0.0;
        for j in i + 1..n {
            let v = 0.5 * (matrix[i * n + j] + matrix[j * n + i]);
            matrix[i * n + j] = v;
            matrix[j * n + i] = v;
        }
    }
    for i in 0..n {
        let row = &mut matrix[i * n..(i + 1) * n];
        let degree: f32 = row.iter().map(|v| v.abs()).sum();
        row.iter_mut().for_each(|v| *v = -*v);
        row[i] = degree;
    }
}

const RESTARTS: usize = 10;
const MAX_ITERATIONS: usize = 300;

fn kmeans(points: &[f32], n: usize, d: usize, k: usize) -> Vec<usize> {
    if n == 0 || k <= 1 {
        return vec![0; n];
    }
    if n <= k {
        return (0..n).collect();
    }
    let mut rng = SplitMix64(0);
    let mut best = (Vec::new(), f32::INFINITY);
    for _ in 0..RESTARTS {
        let (labels, inertia) = lloyd(points, n, d, k, &mut rng);
        if inertia < best.1 {
            best = (labels, inertia);
        }
    }
    best.0
}

fn distance(points: &[f32], i: usize, centers: &[f32], c: usize, d: usize) -> f32 {
    (0..d).map(|j| (points[i * d + j] - centers[c * d + j]).powi(2)).sum()
}

fn lloyd(points: &[f32], n: usize, d: usize, k: usize, rng: &mut SplitMix64) -> (Vec<usize>, f32) {
    let mut centers = seed(points, n, d, k, rng);
    let mut labels = vec![usize::MAX; n];
    let mut near = vec![0f32; n];
    let mut inertia = f32::INFINITY;
    for _ in 0..MAX_ITERATIONS {
        let mut changed = false;
        let mut total = 0f32;
        for i in 0..n {
            let mut best = (0, f32::INFINITY);
            for c in 0..k {
                let dist = distance(points, i, &centers, c, d);
                if dist < best.1 {
                    best = (c, dist);
                }
            }
            if labels[i] != best.0 {
                labels[i] = best.0;
                changed = true;
            }
            near[i] = best.1;
            total += best.1;
        }
        inertia = total;
        if !changed {
            break;
        }
        let mut sums = vec![0f32; k * d];
        let mut counts = vec![0usize; k];
        for i in 0..n {
            counts[labels[i]] += 1;
            for j in 0..d {
                sums[labels[i] * d + j] += points[i * d + j];
            }
        }
        for c in 0..k {
            if counts[c] == 0 {
                let far = (0..n).max_by(|a, b| near[*a].total_cmp(&near[*b])).unwrap();
                sums[c * d..(c + 1) * d].copy_from_slice(&points[far * d..(far + 1) * d]);
                counts[c] = 1;
                near[far] = 0.0;
            }
            let scale = 1.0 / counts[c] as f32;
            for j in 0..d {
                centers[c * d + j] = sums[c * d + j] * scale;
            }
        }
    }
    (labels, inertia)
}

fn seed(points: &[f32], n: usize, d: usize, k: usize, rng: &mut SplitMix64) -> Vec<f32> {
    let mut centers = vec![0f32; k * d];
    let first = (rng.next() % n as u64) as usize;
    centers[..d].copy_from_slice(&points[first * d..(first + 1) * d]);
    let mut min_dist = vec![f32::INFINITY; n];
    for c in 1..k {
        for i in 0..n {
            min_dist[i] = min_dist[i].min(distance(points, i, &centers, c - 1, d));
        }
        let total: f32 = min_dist.iter().sum();
        let mut pick = n - 1;
        if total > 0.0 {
            let target = ((rng.next() >> 11) as f64 * (1.0 / (1u64 << 53) as f64)) as f32 * total;
            let mut acc = 0f32;
            for (i, v) in min_dist.iter().enumerate() {
                acc += v;
                if acc >= target {
                    pick = i;
                    break;
                }
            }
        }
        centers[c * d..(c + 1) * d].copy_from_slice(&points[pick * d..(pick + 1) * d]);
    }
    centers
}

struct SplitMix64(u64);

impl SplitMix64 {
    fn next(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9E37_79B9_7F4A_7C15);
        let mut z = self.0;
        z = (z ^ (z >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
        z ^ (z >> 31)
    }
}
