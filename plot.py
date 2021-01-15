#!/usr/bin/env python3

import argparse
import json

from matplotlib import pyplot as plt
import numpy as np
from sklearn import linear_model


def main():
    args = parse_args()
    data = load_data(args.input_filename)
    plot_data(data, args.title, args.output_filename)


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--title", default="Ford Focus 2015+", help="graph title")
    parser.add_argument(
        "--input-filename", default="car_info.json", help="input filename"
    )
    parser.add_argument(
        "--output-filename", default="ford_focus.png", help="output filename"
    )

    return parser.parse_args()


def load_data(filename):
    with open(filename) as f:
        return json.load(f)


def plot_data(data, title, output_filename):
    xs = np.array([car["mileage"] for car in data]).reshape((-1, 1))
    ys = np.array([car["price"] for car in data])

    model = linear_model.LinearRegression().fit(xs, ys)
    r_sq = model.score(xs, ys)
    print("coefficient of determination:", r_sq)
    print("intercept:", model.intercept_)
    print("slope:", model.coef_)

    model_ys = model.predict(xs)

    fig, ax = plt.subplots()

    ax.scatter(xs, ys, marker="+")
    ax.plot(xs, model_ys, "r")
    ax.set(
        xlabel="mileage", ylabel="price / GBP", title=title,
    )
    ax.grid()

    fig.savefig(output_filename)
    plt.show()


if __name__ == "__main__":
    main()
