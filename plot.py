#!/usr/bin/env python3

import json

from matplotlib import pyplot as plt
import numpy as np
from sklearn import linear_model

INPUT_FILE = "car_info.json"


def main():
    data = load_data()
    plot_data(data)


def load_data():
    with open(INPUT_FILE) as f:
        return json.load(f)


def plot_data(data):
    xs = np.array([car["mileage"] for car in data]).reshape((-1, 1))
    ys = np.array([car["price"] for car in data])

    model = linear_model.LinearRegression().fit(xs, ys)
    r_sq = model.score(xs, ys)
    print("coefficient of determination:", r_sq)
    print("intercept:", model.intercept_)
    print("slope:", model.coef_)

    model_ys = model.predict(xs)

    fig, ax = plt.subplots()

    ax.scatter(xs, ys)
    ax.plot(xs, model_ys, "r")
    ax.set(
        xlabel="mileage", ylabel="price", title="Ford Focus 2015+",
    )
    ax.grid()

    fig.savefig("ford_focus.png")
    plt.show()


if __name__ == "__main__":
    main()
